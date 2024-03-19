package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/Rodert/jupiter-go/jupiter"
	"github.com/Rodert/jupiter-go/solana"
	"log"
)

// go run main.go -userPublicKey userPublicKey -walletPrivateKey walletPrivateKey -input So11111111111111111111111111111111111111112 -output EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v -amount 10000 -slip 250
func main() {
	ctx := context.TODO()
	// 定义一个命令行参数
	userPublicKey := flag.String("userPublicKey", "your userPublicKey", "userPublicKey")
	walletPrivateKey := flag.String("walletPrivateKey", "your walletPrivateKey", "walletPrivateKey")
	input := flag.String("input", "So11111111111111111111111111111111111111112", "输入代币地址，default solana")
	output := flag.String("output", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "输出代币地址，default USDC")
	amount := flag.Int("amount", 10000, "amount 兑换额度，兑换代笔的最小单位")
	slip := flag.Int("slip", 250, "slip 滑点, 兑换代笔的最小单位")
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
	}

	swap := GetSwapJson(*input, *output, *userPublicKey, *amount, *slip)

	signedTx, err := RunSwap(ctx, solanaClient, swap)
	if err != nil {
		fmt.Printf("err: %v", err)
	}
	fmt.Println(signedTx)

	for {
		// wait a bit to let the transaction propagate to the network
		// this is just an example and not a best practice
		// you could use a ticker or wait until we implement the WebSocket monitoring ;)
		//time.Sleep(20 * time.Second)

		// Get the status of the transaction (pull the status from the blockchain at intervals
		// until the transaction is confirmed)
		var confirmed bool
		var err2 error
		//for {
		//defer func() {
		//	if r := recover(); r != nil {
		//		log.Printf("Recovered from panic: %v", r)
		//		// 程序不会退出，可以继续执行其他代码
		//	}
		//}()
		confirmed, err2 = solanaClient.CheckSignature(ctx, signedTx)
		if err2 != nil {
			//panic(err)
			fmt.Printf("err2: %+v", err2)
		}
		//}
		fmt.Printf("confirmed: %+v\n", confirmed)
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
	fmt.Printf("%+v", swap)
	return swap
}
