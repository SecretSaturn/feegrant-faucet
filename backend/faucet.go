package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/dpapathanasiou/go-recaptcha"
	"github.com/joho/godotenv"
	"github.com/tendermint/tmlibs/bech32"
	"github.com/tomasen/realip"
)

var chain string
var recaptchaSecretKey string
var amountFaucet string
var key string
var node string
var publicURL string
var gasPrices string
var gas string

type claimStruct struct {
	Address  string
	Response string
}

func getEnv(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		fmt.Println("Found", key)
		return value
	}

	log.Fatal("Error loading environment variable: ", key)
	return ""
}

func main() {
	err := godotenv.Load(".env.local", ".env")
	if err != nil {
		log.Fatal("Error loading .env file", err)
	}

	chain = getEnv("FAUCET_CHAIN")
	recaptchaSecretKey = getEnv("FAUCET_RECAPTCHA_SECRET_KEY")
	amountFaucet = getEnv("FAUCET_AMOUNT_FAUCET")
	key = getEnv("FAUCET_KEY")
	node = getEnv("FAUCET_NODE")
	publicURL = getEnv("FAUCET_PUBLIC_URL")
	localStr := getEnv("LOCAL_RUN")
	gasPrices = getEnv("GAS_PRICES")
	gas = getEnv("GAS")

	recaptcha.Init(recaptchaSecretKey)

	fs := http.FileServer(http.Dir("dist"))
	http.Handle("/", fs)

	http.HandleFunc("/claim", getCoinsHandler)

	localBool, err := strconv.ParseBool(localStr)
	if err != nil {
		log.Fatal("Failed to parse dotenv var: LOCAL_RUN", err)
	} else if !localBool {
		if err := http.ListenAndServe(publicURL, nil); err != nil {
			log.Fatal("failed to start server", err)
		}
	} else {
		if err := http.ListenAndServe(publicURL, nil); err != nil {
			log.Fatal("failed to start server", err)
		}
	}

}

func executeCmd(command string) (e error) {
	cmd, stdout, _ := goExecute(command)

	var txOutput struct {
		Height string
		Txhash string
		RawLog string
	}

	output := ""
	buf := bufio.NewReader(stdout)
	for {
		line, _, err := buf.ReadLine()
		if err != nil {
			break
		}

		output += string(line)
	}

	if err := json.Unmarshal([]byte(output), &txOutput); err != nil {
		fmt.Printf("Error: '%s' for parsing the following: '%s'\n", err, output)
		return fmt.Errorf("server error. can't send tokens")
	}

	fmt.Println("Granted Fee. txhash:", txOutput.Txhash)

	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}

	return nil
}

func goExecute(command string) (cmd *exec.Cmd, pipeOut io.ReadCloser, pipeErr io.ReadCloser) {
	cmd = getCmd(command)
	pipeOut, _ = cmd.StdoutPipe()
	pipeErr, _ = cmd.StderrPipe()

	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	return cmd, pipeOut, pipeErr
}

func getCmd(command string) *exec.Cmd {
	// split command into command and args
	split := strings.Split(command, " ")

	var cmd *exec.Cmd
	if len(split) == 1 {
		cmd = exec.Command(split[0])
	} else {
		cmd = exec.Command(split[0], split[1:]...)
	}

	return cmd
}

func getCoinsHandler(w http.ResponseWriter, request *http.Request) {
	var claim claimStruct

	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "*")

	if request.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if request.Method != http.MethodPost {
		http.Error(w, "Only POST allowed.", http.StatusBadRequest)
		return
	}

	// decode JSON response from front end
	decoder := json.NewDecoder(request.Body)
	decoderErr := decoder.Decode(&claim)
	if decoderErr != nil {
		panic(decoderErr)
	}

	// make sure address is bech32
	readableAddress, decodedAddress, decodeErr := bech32.DecodeAndConvert(claim.Address)
	if decodeErr != nil {
		panic(decodeErr)
	}
	// re-encode the address in bech32
	encodedAddress, encodeErr := bech32.ConvertAndEncode(readableAddress, decodedAddress)
	if encodeErr != nil {
		panic(encodeErr)
	}

	// make sure captcha is valid
	clientIP := realip.FromRequest(request)
	captchaResponse := claim.Response
	captchaPassed, captchaErr := recaptcha.Confirm(clientIP, captchaResponse)
	if captchaErr != nil {
		panic(captchaErr)
	}

	// send the coins!
	if captchaPassed {
		//sendFaucet := fmt.Sprintf(
		//"secretcli tx bank send %v %v %v --gas-prices=%v --chain-id=%v --node=%v --output=json -y --gas 15000",
		//	key, encodedAddress, amountFaucet, gasPrices, chain, node)
		t := time.Now().AddDate(0, 0, 1)
		expiration := t.Format(time.RFC3339)
		sendFaucet := fmt.Sprintf(
			"secretcli tx feegrant grant %v %v --gas-prices=%v --spend-limit=%v --expiration=%v --chain-id=%v --node=%v --output=json -y --gas %v",
			key, encodedAddress, gasPrices, amountFaucet, expiration, chain, node, gas)
		fmt.Println(time.Now().UTC().Format(time.RFC3339), encodedAddress, "[1]")
		fmt.Println("Executing cmd:", sendFaucet)
		err := executeCmd(sendFaucet)

		// If command fails, reutrn an error
		if err != nil {
			fmt.Println("Error executing command:", err)
			http.Error(w, err.Error(), 500)
		}
	}

	return
}
