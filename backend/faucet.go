package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/tendermint/tmlibs/bech32"
)

var cli_name string
var chain string
var amountFaucet string
var denomFaucet string
var key string
var rpc_node string
var lcd_node string
var faucet_addr string
var publicURL string
var gasPriceAmount string
var gasPriceDenom string
var gasRevoke string
var gasGrant string
var memo string

type claimStruct struct {
	Address  string
	Response string
}

type AllowanceJSON struct {
	Allowance struct {
		Granter   string `json:"granter"`
		Grantee   string `json:"grantee"`
		Allowance struct {
			Type       string `json:"@type"`
			SpendLimit []struct {
				Denom  string `json:"denom"`
				Amount string `json:"amount"`
			} `json:"spend_limit"`
			Expiration time.Time `json:"expiration"`
		} `json:"allowance"`
	} `json:"allowance"`
}

type AccountsJSON struct {
	Height string `json:"height"`
	Result struct {
		Type  string `json:"type"`
		Value struct {
			Address   string `json:"address"`
			PublicKey struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"public_key"`
			AccountNumber string `json:"account_number"`
			Sequence      string `json:"sequence"`
		} `json:"value"`
	} `json:"result"`
}

type ErrorJSON struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Details []interface{} `json:"details"`
}

type txOutput struct {
	Height string
	Txhash string
	RawLog string
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
	cli_name = getEnv("CLI_NAME")
	amountFaucet = getEnv("FAUCET_AMOUNT")
	denomFaucet = getEnv("FAUCET_DENOM")
	key = getEnv("FAUCET_KEY")
	rpc_node = getEnv("RPC_NODE")
	lcd_node = getEnv("LCD_NODE")
	faucet_addr = getEnv("FAUCET_ADDRESS")
	publicURL = getEnv("FAUCET_PUBLIC_URL")
	localStr := getEnv("LOCAL_RUN")
	gasPriceAmount = getEnv("GAS_PRICE_AMOUNT")
	gasPriceDenom = getEnv("GAS_PRICE_DENOM")
	gasRevoke = getEnv("GAS_REVOKE")
	gasGrant = getEnv("GAS_GRANT")
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

func executeCmd(encodedAddress string, AllowanceType string) (e error) {
	cmd, stdout, _ := goExecute(encodedAddress, AllowanceType)

	output := ""
	buf := bufio.NewReader(stdout)
	for {
		line, _, err := buf.ReadLine()
		if err != nil {
			break
		}

		output += string(line)
	}

	txOut := txOutput{}

	if err := json.Unmarshal([]byte(output), &txOut); err != nil {
		fmt.Printf("Error: '%s' for parsing the following: '%s'\n", err, output)
		return fmt.Errorf("server error. can't send tokens")
	}

	fmt.Println("Granted Fee. txhash:", txOut.Txhash)

	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}

	return nil
}

func goExecute(encodedAddress string, AllowanceType string) (cmd *exec.Cmd, pipeOut io.ReadCloser, pipeErr io.ReadCloser) {

	cmd = getCmd(encodedAddress, AllowanceType)

	pipeOut, _ = cmd.StdoutPipe()
	pipeErr, _ = cmd.StderrPipe()

	err := cmd.Start()
	if err != nil || cmd == nil {
		log.Fatal(err)
	}

	return cmd, pipeOut, pipeErr
}

func getCmd(encodedAddress string, AllowanceType string) *exec.Cmd {

	t := time.Now().AddDate(0, 0, 1)
	expiration := t.Format(time.RFC3339)

	if AllowanceType == "grant" {

		gasGrantInt, err1 := strconv.ParseFloat(gasGrant, 64)
		gasPriceAmountInt, err2 := strconv.ParseFloat(gasPriceAmount, 64)

		if err1 != nil || err2 != nil {
			log.Fatal(err1, err2)
		}

		mulResult := gasGrantInt * gasPriceAmountInt
		mulResultStr := strconv.Itoa(int(mulResult))

		var command = fmt.Sprintf("echo '{\"body\":{\"messages\":[{\"@type\":\"/cosmos.feegrant.v1beta1.MsgGrantAllowance\",\"granter\":\"%v\",\"grantee\":\"%v\",\"allowance\":{\"@type\":\"/cosmos.feegrant.v1beta1.BasicAllowance\",\"spend_limit\":[{\"denom\":\"%v\",\"amount\":\"%v\"}],\"expiration\":\"%v\"}}],\"memo\":\"%v\",\"timeout_height\":\"0\",\"extension_options\":[],\"non_critical_extension_options\":[]},\"auth_info\":{\"signer_infos\":[],\"fee\":{\"amount\":[{\"denom\":\"uscrt\",\"amount\":\"%v\"}],\"gas_limit\":\"%v\",\"payer\":\"\",\"granter\":\"\"}},\"signatures\":[]}' | sudo %v tx sign - --from=%v --chain-id=%v --output=json --keyring-backend=test | %v tx broadcast - --node=%v", faucet_addr, encodedAddress, denomFaucet, amountFaucet, expiration, memo, mulResultStr, gasGrant, cli_name, key, chain, cli_name, rpc_node)

		fmt.Println(time.Now().UTC().Format(time.RFC3339), encodedAddress, "[1]")
		fmt.Println("Executing cmd:", command)

		var cmd *exec.Cmd
		cmd = exec.Command("bash", "-c", command)

		return cmd

	} else if AllowanceType == "revoke+grant" {

		gasGrantInt, err1 := strconv.ParseFloat(gasGrant, 64)
		gasPriceAmountInt, err2 := strconv.ParseFloat(gasPriceAmount, 64)

		if err1 != nil || err2 != nil {
			log.Fatal(err1, err2)
		}

		mulResult := gasGrantInt * gasPriceAmountInt
		mulResultStr := strconv.Itoa(int(mulResult))

		var command = fmt.Sprintf("echo '{\"body\":{\"messages\":[{\"@type\":\"/cosmos.feegrant.v1beta1.MsgRevokeAllowance\",\"granter\":\"%v\",\"grantee\":\"%v\"},{\"@type\":\"/cosmos.feegrant.v1beta1.MsgGrantAllowance\",\"granter\":\"%v\",\"grantee\":\"%v\",\"allowance\":{\"@type\":\"/cosmos.feegrant.v1beta1.BasicAllowance\",\"spend_limit\":[{\"denom\":\"%v\",\"amount\":\"%v\"}],\"expiration\":\"%v\"}}],\"memo\":\"%v\",\"timeout_height\":\"0\",\"extension_options\":[],\"non_critical_extension_options\":[]},\"auth_info\":{\"signer_infos\":[],\"fee\":{\"amount\":[{\"denom\":\"uscrt\",\"amount\":\"%v\"}],\"gas_limit\":\"%v\",\"payer\":\"\",\"granter\":\"\"}},\"signatures\":[]}' | sudo %v tx sign - --from=%v --chain-id=%v --output=json --keyring-backend=test | %v tx broadcast - --node=%v", faucet_addr, encodedAddress, faucet_addr, encodedAddress, denomFaucet, amountFaucet, expiration, memo, mulResultStr, gasGrant, cli_name, key, chain, cli_name, rpc_node)

		fmt.Println(time.Now().UTC().Format(time.RFC3339), encodedAddress, "[1]")
		fmt.Println("Executing cmd:", command)

		var cmd *exec.Cmd
		cmd = exec.Command("bash", "-c", command)

		return cmd
	}

	return nil
}

func queryNode(w http.ResponseWriter, query string) (body []byte) {

	url := lcd_node + query

	httpClient := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}

	req, reqErr := http.NewRequest(http.MethodGet, url, nil)
	if reqErr != nil {
		fmt.Println("Error executing command:", reqErr)
		http.Error(w, reqErr.Error(), 500)
		return nil
	}

	res, getErr := httpClient.Do(req)
	if getErr != nil {
		fmt.Println("Error executing command:", getErr)
		http.Error(w, getErr.Error(), 500)
		return nil
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)

	if readErr != nil {
		fmt.Println("Error executing command:", readErr)
		http.Error(w, readErr.Error(), 500)
		return nil
	}
	return body
}

func getCoinsHandler(w http.ResponseWriter, request *http.Request) {

	var claim claimStruct

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

	//check if a fee grant exists

	query := "/cosmos/feegrant/v1beta1/allowance/" + faucet_addr + "/" + encodedAddress

	body := queryNode(w, query)

	if body == nil {
		return
	}

	allowanceJSON := AllowanceJSON{}

	jsonErr := json.Unmarshal(body, &allowanceJSON)

	errorJSON := ErrorJSON{}

	parsedDate := allowanceJSON.Allowance.Allowance.Expiration

	fmt.Println(parsedDate)

	if jsonErr != nil || parsedDate.IsZero() {
		jsonErr := json.Unmarshal(body, &errorJSON)

		if ((errorJSON.Code == 13) && (errorJSON.Message == "fee-grant not found: unauthorized")) || parsedDate.IsZero() {

			fmt.Println("No active Fee Grant")

			grantErr := executeCmd(encodedAddress, "grant")

			// If command fails, return an error
			if grantErr != nil {
				fmt.Println("Error executing command:", grantErr)
				http.Error(w, grantErr.Error(), 500)
				return
			}
			return
		}
		if jsonErr != nil {
			fmt.Println("Error executing command:", jsonErr)
			http.Error(w, jsonErr.Error(), 500)
			return
		}
	}

	if time.Now().After(parsedDate) {

		fmt.Println("Existing Fee Grant expired")

		revokeErr := executeCmd(encodedAddress, "revoke+grant")

		// If command fails, return an error
		if revokeErr != nil {
			fmt.Println("Error executing command:", revokeErr)
			http.Error(w, revokeErr.Error(), 500)
			return
		}

	} else {
		fmt.Println("Existing Fee Grant did not expire")
		http.Error(w, "Existing Fee Grant did not expire", 500)
		return
	}
	return
}
