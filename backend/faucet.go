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

func executeCmd(encodedAddress string, AllowanceType string, sequence int) (e error) {
	cmd, stdout, _ := goExecute(encodedAddress, AllowanceType, sequence)

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

func goExecute(encodedAddress string, AllowanceType string, sequence int) (cmd *exec.Cmd, pipeOut io.ReadCloser, pipeErr io.ReadCloser) {
	cmd = getCmd(encodedAddress, AllowanceType, sequence)

	pipeOut, _ = cmd.StdoutPipe()
	pipeErr, _ = cmd.StderrPipe()

	err := cmd.Start()
	if err != nil || cmd == nil {
		log.Fatal(err)
	}

	return cmd, pipeOut, pipeErr
}

func getCmd(encodedAddress string, AllowanceType string, sequence int) *exec.Cmd {
	if AllowanceType == "grant" {
		t := time.Now().AddDate(0, 0, 1)
		expiration := t.Format(time.RFC3339)

		var command []string

		command = append(command, "secretcli")
		command = append(command, "tx")
		command = append(command, "feegrant")
		command = append(command, "grant")
		command = append(command, key)
		command = append(command, encodedAddress)
		command = append(command, fmt.Sprintf("--gas-prices=%v", gasPrices))
		command = append(command, fmt.Sprintf("--spend-limit=%v", amountFaucet))
		command = append(command, fmt.Sprintf("--expiration=%v", expiration))
		command = append(command, fmt.Sprintf("--chain-id=%v", chain))
		command = append(command, fmt.Sprintf("--node=%v", node))
		command = append(command, "--output=json")
		command = append(command, "-y")
		command = append(command, fmt.Sprintf("--gas=%v", gasGrant))
		command = append(command, "--keyring-backend=test")
		command = append(command, fmt.Sprintf("--note=%v", memo))

		if sequence != 0 {
			command = append(command, fmt.Sprintf("--sequence=%v", sequence))
		}

		fmt.Println(time.Now().UTC().Format(time.RFC3339), encodedAddress, "[1]")
		fmt.Println("Executing cmd:", strings.Join(command[:], " "))

		var cmd *exec.Cmd
		cmd = exec.Command(command[0], command[1:]...)

		return cmd

	} else if AllowanceType == "revoke" {

		var command []string

		command = append(command, "secretcli")
		command = append(command, "tx")
		command = append(command, "feegrant")
		command = append(command, "revoke")
		command = append(command, key)
		command = append(command, encodedAddress)
		command = append(command, fmt.Sprintf("--gas-prices=%v", gasPrices))
		command = append(command, fmt.Sprintf("--chain-id=%v", chain))
		command = append(command, fmt.Sprintf("--node=%v", node))
		command = append(command, "--output=json")
		command = append(command, "-y")
		command = append(command, fmt.Sprintf("--gas=%v", gasRevoke))
		command = append(command, "--keyring-backend=test")

		if sequence != 0 {
			command = append(command, fmt.Sprintf("--sequence=%v", sequence))
		}

		fmt.Println(time.Now().UTC().Format(time.RFC3339), encodedAddress, "[1]")
		fmt.Println("Executing cmd:", strings.Join(command[:], " "))

		var cmd *exec.Cmd
		cmd = exec.Command(command[0], command[1:]...)

		return cmd

	} else {
		return nil
	}

	return nil
}

func queryNode(w http.ResponseWriter, query string) (body []byte) {

	url := "http://lcd.mainnet.secretsaturn.net/" + query

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

	query := "cosmos/feegrant/v1beta1/allowance/secret1tq6y8waegggp4fv2fcxk3zmpsmlfadyc7lsd69/" + encodedAddress

	body := queryNode(w, query)

	if body == nil {
		return
	}

	allowanceJSON := AllowanceJSON{}

	jsonErr := json.Unmarshal(body, &allowanceJSON)

	errorJSON := ErrorJSON{}

	if jsonErr != nil || allowanceJSON.Allowance.Allowance.Expiration.IsZero() {
		jsonErr := json.Unmarshal(body, &errorJSON)

		if (errorJSON.Code == 2) && (errorJSON.Message == "rpc error: code = Internal desc = fee-grant not found: unauthorized: unknown request") {
			fmt.Println("No active Fee Grant")

			grantErr := executeCmd(encodedAddress, "grant", 0)

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

	parsedDate := allowanceJSON.Allowance.Allowance.Expiration

	if time.Now().After(parsedDate) {
		fmt.Println("Existing Fee Grant expired")

		query := "auth/accounts/" + "secret1tq6y8waegggp4fv2fcxk3zmpsmlfadyc7lsd69"

		body := queryNode(w, query)

		if body == nil {
			return
		}

		accountsJSON := AccountsJSON{}

		jsonErr := json.Unmarshal(body, &accountsJSON)

		if jsonErr != nil {
			fmt.Println("Error executing command:", jsonErr)
			http.Error(w, jsonErr.Error(), 500)
			return
		}

		sequence, convErr := strconv.Atoi(accountsJSON.Result.Value.Sequence)

		if convErr != nil {
			fmt.Println("Error executing command:", convErr)
			http.Error(w, convErr.Error(), 500)
			return
		}

		revokeErr := executeCmd(encodedAddress, "revoke", sequence)

		// If command fails, reutrn an error
		if revokeErr != nil {
			fmt.Println("Error executing command:", revokeErr)
			http.Error(w, revokeErr.Error(), 500)
			return
		}

		grantErr := executeCmd(encodedAddress, "grant", sequence+1)

		// If command fails, reutrn an error
		if grantErr != nil {
			fmt.Println("Error executing command:", grantErr)
			http.Error(w, grantErr.Error(), 500)
			return
		}

	} else {
		fmt.Println("Existing Fee Grant did not expire")
		http.Error(w, "Existing Fee Grant did not expire", 500)
		return
	}
	return
}
