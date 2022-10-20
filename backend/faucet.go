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

	"github.com/joho/godotenv"
	"github.com/tendermint/tmlibs/bech32"
)

var chain string
var amountFaucet string
var key string
var node string
var publicURL string
var gasPrices string
var gas string
var memo string

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
	amountFaucet = getEnv("FAUCET_AMOUNT_FAUCET")
	key = getEnv("FAUCET_KEY")
	node = getEnv("FAUCET_NODE")
	publicURL = getEnv("FAUCET_PUBLIC_URL")
	localStr := getEnv("LOCAL_RUN")
	gasPrices = getEnv("GAS_PRICES")
	gas = getEnv("GAS")
	memo = getEnv("MEMO")

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

func executeCmd(encodedAddress string) (e error) {
	cmd, stdout, _ := goExecute(encodedAddress)

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

func goExecute(encodedAddress string) (cmd *exec.Cmd, pipeOut io.ReadCloser, pipeErr io.ReadCloser) {
	cmd = getCmd(encodedAddress)

	pipeOut, _ = cmd.StdoutPipe()
	pipeErr, _ = cmd.StderrPipe()

	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	return cmd, pipeOut, pipeErr
}

func getCmd(encodedAddress string) *exec.Cmd {

	t := time.Now().AddDate(0, 0, 1)
	expiration := t.Format(time.RFC3339)

	var command [16]string

	command[0] = "secretcli"
	command[1] = "tx"
	command[2] = "feegrant"
	command[3] = "grant"
	command[4] = key
	command[5] = encodedAddress
	command[6] = fmt.Sprintf("--gas-prices=%v", gasPrices)
	command[7] = fmt.Sprintf("--spend-limit=%v", amountFaucet)
	command[8] = fmt.Sprintf("--expiration=%v", expiration)
	command[9] = fmt.Sprintf("--chain-id=%v", chain)
	command[10] = fmt.Sprintf("--node=%v", node)
	command[11] = "--output=json"
	command[12] = "-y"
	command[13] = fmt.Sprintf("--gas=%v", gas)
	command[14] = "--keyring-backend=test"
	command[15] = fmt.Sprintf("--note=%v", memo)

	fmt.Println(time.Now().UTC().Format(time.RFC3339), encodedAddress, "[1]")
	fmt.Println("Executing cmd:", strings.Join(command[:], " "))

	var cmd *exec.Cmd
	cmd = exec.Command(command[0], command[1:]...)

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
		fmt.Println("Error executing command:", decoderErr)
		http.Error(w, decoderErr.Error(), 500)
		return
	}

	// make sure address is bech32
	readableAddress, decodedAddress, decodeErr := bech32.DecodeAndConvert(claim.Address)
	if decodeErr != nil {
		fmt.Println("Error executing command:", decodeErr)
		http.Error(w, decodeErr.Error(), 500)
		return
	}
	// re-encode the address in bech32
	encodedAddress, encodeErr := bech32.ConvertAndEncode(readableAddress, decodedAddress)
	if encodeErr != nil {
		fmt.Println("Error executing command:", encodeErr)
		http.Error(w, encodeErr.Error(), 500)
		return
	}

	err := executeCmd(encodedAddress)

	// If command fails, reutrn an error
	if err != nil {
		fmt.Println("Error executing command:", err)
		http.Error(w, err.Error(), 500)
	}

	return
}
