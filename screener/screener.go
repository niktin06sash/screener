package screener

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func mustClient(url string, ctx context.Context) (*ethclient.Client, error) {
	client, err := ethclient.DialContext(ctx, url)
	if err != nil {
		log.Printf("Failed to connect to node %s: %v", url, err)
		return nil, err
	}
	return client, nil
}

type TokenData struct {
	TInfo  TokenInfo
	RInfo  ReceiverInfo
	BlInfo BlockChainInfo
}

type ReceiverInfo struct {
	Address common.Address
}
type BlockChainInfo struct {
	TxHash string
	Block  uint64
}
type TokenInfo struct {
	Address common.Address
}
type ScreenerConfig struct {
	chainID        *big.Int
	clients        []*ethclient.Client
	node           int
	Tokenchan      chan TokenData
	watchedsWallet []common.Hash
	txMap          map[common.Hash]bool
	Token_count    map[string]int
}

func getWatchedWallets() []common.Hash {
	walletsTxt := "wallets1.txt"
	pbs := []common.Hash{}
	txtData, err := os.ReadFile(walletsTxt)
	if err != nil {
		log.Println(err)
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(txtData)), "\n")

	for i := 0; i < len(lines); i++ {
		pubKey := strings.TrimSpace(lines[i])
		addr := common.HexToHash(pubKey)
		pbs = append(pbs, addr)
		log.Println(addr)
	}
	return pbs
}
func getWatchedWalletsOS() []common.Hash {
	walletNum := os.Getenv("WALLET_NUM")
	num, _ := strconv.Atoi(walletNum)
	pbs := []common.Hash{}
	for i := 1; i <= num; i++ {
		pubKey := os.Getenv(fmt.Sprintf("WATCHED_WALLET_%s", strconv.Itoa(i)))
		addr := common.HexToHash(pubKey)
		pbs = append(pbs, addr)
	}
	return pbs
}
func getNodesOS() []string {
	nodesNum := os.Getenv("NODES_NUM")
	num, _ := strconv.Atoi(nodesNum)
	pbs := []string{}
	for i := 2; i <= num; i++ {
		node := os.Getenv(fmt.Sprintf("quick%swss_base", strconv.Itoa(i)))
		pbs = append(pbs, node)
	}
	return pbs
}
func NewScreenerConfig(ctx context.Context) (*ScreenerConfig, error) {
	nodes := getNodesOS()
	clients := make([]*ethclient.Client, len(nodes))
	for i := 0; i < len(clients); i++ {
		client, err := mustClient(nodes[i], ctx)
		if err != nil {
			return nil, err
		}
		clients[i] = client
	}
	chainID, err := clients[0].NetworkID(ctx)
	if err != nil {
		log.Printf("Failed to get chainID: %v", err)
		return nil, err
	}
	tokenDataChan := make(chan TokenData, 1000)
	watcheds := getWatchedWalletsOS()
	str := &ScreenerConfig{
		chainID: chainID,

		Tokenchan:      tokenDataChan,
		watchedsWallet: watcheds,
		clients:        clients,
		txMap:          make(map[common.Hash]bool),
		Token_count:    make(map[string]int),
	}
	return str, nil
}
func (scr *ScreenerConfig) ScreenerReader(ctx context.Context) error {

	query := ethereum.FilterQuery{
		Topics: [][]common.Hash{
			{common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")},
			{},
			scr.watchedsWallet,
		},
	}

	logsChan := make(chan types.Log, 1000)
	sub, err := scr.clients[scr.node].SubscribeFilterLogs(ctx, query, logsChan)
	if err != nil {
		log.Printf("Failed to subscribe to logs: %v", err)
		return err
	}
	log.Println("Listening for TransferToken events...")
	go func() {
		defer func() {
			sub.Unsubscribe()
			close(logsChan)
			close(scr.Tokenchan)
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-sub.Err():
				log.Printf("Error from subscription on logs: %v", err)
				scr.node = (scr.node + 1) % len(scr.clients)
				log.Printf("Switch to node: %d", scr.node)
				sub.Unsubscribe()
				newSub, err := scr.clients[scr.node].SubscribeFilterLogs(ctx, query, logsChan)
				if err != nil {
					log.Printf("Failed to subscribe to logs: %v", err)
					continue
				}
				sub = newSub
			case vLog := <-logsChan:
				token := vLog.Address
				if scr.txMap[vLog.TxHash] {
					continue
				}
				log.Println(token, vLog.TxHash)
				if token != common.HexToAddress("0x55d398326f99059fF775485246999027B3197955") && token != common.HexToAddress("0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c") && token != common.HexToAddress("0x2170Ed0880ac9A755fd29B2688956BD959F933F8") {
					to := common.HexToAddress(vLog.Topics[2].Hex())
					tokenData := TokenData{
						TInfo:  TokenInfo{Address: token},
						RInfo:  ReceiverInfo{Address: to},
						BlInfo: BlockChainInfo{TxHash: vLog.TxHash.Hex(), Block: vLog.BlockNumber},
					}
					tx, _, err := scr.clients[scr.node].TransactionByHash(ctx, vLog.TxHash)
					if err != nil {
						log.Printf("Failed to get transaction: %v", err)
						continue
					}
					signer := types.LatestSignerForChainID(scr.chainID)
					sender, err := types.Sender(signer, tx)
					if err != nil {
						log.Printf("Failed to get sender: %v", err)
						continue
					}
					scr.txMap[vLog.TxHash] = true
					if sender != to {
						log.Printf("Spam token!")
						time.Sleep(100 * time.Millisecond)
						continue
					}
					scr.Tokenchan <- tokenData
					time.Sleep(100 * time.Millisecond)
				}
			}
		}
	}()
	return nil
}
