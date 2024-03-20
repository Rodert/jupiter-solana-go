package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Rodert/jupiter-go/jupiter"
	"github.com/Rodert/jupiter-go/solana"
	"github.com/shopspring/decimal"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// go run main.go -userPublicKey userPublicKey -walletPrivateKey walletPrivateKey -input So11111111111111111111111111111111111111112 -output EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v -amount 0.001 -slip 0.001
func main() {
	start := time.Now()
	fmt.Printf("建议开启电脑全局代理。\nusage: %v\n\n", "./solana-letgo-mac -userPublicKey userPublicKey -walletPrivateKey walletPrivateKey -input So11111111111111111111111111111111111111112 -output EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v -amount 0.1 -slip 0.1")
	ctx := context.TODO()
	// 定义一个命令行参数
	userPublicKey := flag.String("userPublicKey", "your userPublicKey", "userPublicKey 钱包公钥。")
	walletPrivateKey := flag.String("walletPrivateKey", "your walletPrivateKey", "walletPrivateKey 钱包私钥。")
	input := flag.String("input", "So11111111111111111111111111111111111111112", "input 输入代币地址，默认 solana 代币。")
	output := flag.String("output", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "output 输出代币地址，默认 USDC 代币。")
	amount := flag.Float64("amount", 0.001, "amount 兑换数额，比如：0.1 solana 代币用于交换，填写 0.1。")
	slip := flag.Float64("slip", 0.01, "slip 滑点, 比如设置 0.1%，填写 0.1")
	//0.1 ～ 0.001
	slippageBps := int(*slip * 100) // 滑点转换
	if slippageBps < 1 {
		return
	}
	decimals, err2 := GetDecimals(*input)
	if err2 != nil {
		fmt.Println(err2.Error())
		return
	}
	stringDecimals := MulStringDecimals(*amount, decimals)
	amountValue, err2 := strconv.Atoi(stringDecimals)
	if err2 != nil {
		fmt.Println(err2.Error())
		return
	}
	flag.PrintDefaults()
	// 解析命令行参数
	flag.Parse()

	// 检查是否有额外的非标志参数
	if flag.NArg() > 0 {
		log.Fatalf("Unexpected arguments: %v\n", flag.Args())
	}

	// Create a wallet from private key
	wallet, err := solana.NewWalletFromPrivateKeyBase58(*walletPrivateKey)
	if err != nil {
		// handle me
	}

	// Create a Solana client
	solanaClient, err := solana.NewClient(wallet, "https://api.mainnet-beta.solana.com")
	if err != nil {
		// handle me
		fmt.Printf("%v\n", err)
		return
	}
	swap := GetSwapJson(*input, *output, *userPublicKey, amountValue, slippageBps)
	fmt.Printf("交易信息： %+v\n", swap)
	log.Printf("执行耗时: %s\n", time.Now().Sub(start))

	signedTx, err := RunSwap(ctx, solanaClient, swap)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	fmt.Printf("交易哈希： %v\n", signedTx)
	log.Printf("交易已提交，执行耗时: %s\n", time.Now().Sub(start))

	GetStatus(ctx, solanaClient, signedTx)
	log.Printf("交易已上链，执行耗时: %s\n", time.Now().Sub(start))
}

// 交易状态
func GetStatus(ctx context.Context, solanaClient solana.Client, signedTx solana.TxID) {
	for {
		// wait a bit to let the transaction propagate to the network
		// this is just an example and not a best practice
		// you could use a ticker or wait until we implement the WebSocket monitoring ;)
		//time.Sleep(20 * time.Second)

		// Get the status of the transaction (pull the status from the blockchain at intervals
		// until the transaction is confirmed)
		var confirmed bool
		var err2 error
		confirmed, err2 = solanaClient.CheckSignature(ctx, signedTx)
		if err2 != nil {
			//panic(err)
			fmt.Printf("pinding： %+v\n", err2)
		} else {
			fmt.Printf("是否完成: %+v\n", confirmed)
			return
		}
		time.Sleep(2 * time.Second)
	}
}

func RunSwap(ctx context.Context, solanaClient solana.Client, swap *jupiter.SwapResponse) (solana.TxID, error) {
	// ... previous code
	// swap := swapResponse.JSON200

	// Sign and send the transaction
	signedTx, err := solanaClient.SendTransactionOnChain(ctx, swap.SwapTransaction)
	if err != nil {
		// handle me
	}
	return signedTx, err
}

func GetSwapJson(input, output, userPublicKey string, amount, slip int) *jupiter.SwapResponse {
	jupClient, err := jupiter.NewClientWithResponses(jupiter.DefaultAPIURL)
	if err != nil {
		panic(err)
	}
	ctx := context.TODO()
	slippageBps := slip

	// Get the current quote for a swap
	quoteResponse, err := jupClient.GetQuoteWithResponse(ctx, &jupiter.GetQuoteParams{
		InputMint:   input,
		OutputMint:  output,
		Amount:      amount,
		SlippageBps: &slippageBps,
	})
	if err != nil {
		panic(err)
	}

	if quoteResponse.JSON200 == nil {
		panic("invalid GetQuoteWithResponse response")
	}

	quote := quoteResponse.JSON200

	// More info: https://station.jup.ag/docs/apis/troubleshooting
	prioritizationFeeLamports := jupiter.SwapRequest_PrioritizationFeeLamports{}
	if err = prioritizationFeeLamports.UnmarshalJSON([]byte(`"auto"`)); err != nil {
		panic(err)
	}

	dynamicComputeUnitLimit := true
	// Get instructions for a swap
	swapResponse, err := jupClient.PostSwapWithResponse(ctx, jupiter.PostSwapJSONRequestBody{
		PrioritizationFeeLamports: &prioritizationFeeLamports,
		QuoteResponse:             *quote,
		UserPublicKey:             userPublicKey,
		DynamicComputeUnitLimit:   &dynamicComputeUnitLimit,
	})
	if err != nil {
		panic(err)
	}

	if swapResponse.JSON200 == nil {
		panic("invalid PostSwapWithResponse response")
	}

	swap := swapResponse.JSON200
	//fmt.Printf("%+v", swap)
	return swap
}

func GetDecimals(address string) (int, error) {
	if address == "So11111111111111111111111111111111111111112" {
		return 9, nil
	}

	param := fmt.Sprintf(`{"jsonrpc":"2.0", "id":1, "method":"getTokenSupply", "params": ["%v"]}`, address)
	client := &http.Client{}
	var data = strings.NewReader(param)
	req, err := http.NewRequest("POST", "https://docs-demo.solana-mainnet.quiknode.pro/", data)
	if err != nil {
		//log.Fatal(err)
		return 9, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		//log.Fatal(err)
		return 9, err
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		//log.Fatal(err)
		return 9, err
	}
	//fmt.Printf("%s\n", bodyText)
	var getTokenSupplyResponse GetTokenSupplyResponse
	err = json.Unmarshal(bodyText, &getTokenSupplyResponse)
	if err != nil {
		return 9, err
	}
	return getTokenSupplyResponse.Result.Value.Decimals, nil
}

func MulStringDecimals(amount float64, precision int) string {
	aDecimal := decimal.NewFromFloat(amount)
	// digit, _ := strconv.Atoi(precision)
	power := decimal.NewFromInt(int64(precision)) // 10 的 N 次方
	ten := decimal.NewFromInt(10)
	res := aDecimal.Mul(ten.Pow(power))
	return res.Round(3).String()
}

type GetTokenSupplyResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Result  Result `json:"result"`
	ID      int    `json:"id"`
}
type Context struct {
	APIVersion string `json:"apiVersion"`
	Slot       int    `json:"slot"`
}
type Value struct {
	Amount         string  `json:"amount"`
	Decimals       int     `json:"decimals"`
	UIAmount       float64 `json:"uiAmount"`
	UIAmountString string  `json:"uiAmountString"`
}
type Result struct {
	Context Context `json:"context"`
	Value   Value   `json:"value"`
}
