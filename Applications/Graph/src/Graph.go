package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

var testNum int = 100
var testTreeNum int = 20
var straightTreeNum int = 10

func GetForwardTestSequence() []int {
	var testSeq []int
	for i := straightTreeNum; i < testTreeNum; i++ {
		testSeq = append(testSeq, i*testNum)
	}
	return testSeq
}

func GetBackwardTestSequence() []int {
	var testSeq []int
	for i := 0; i < straightTreeNum; i++ {
		testSeq = append(testSeq, (i+1)*testNum-1)
	}
	return testSeq
}

// SmartContract provides functions for managing an Record
type SmartContract struct {
	contractapi.Contract
}

// Record describes the provenance node structure
type Record struct {
	ID       string `json:"ID"`
	Previous string `json:"previous"`
	Future   string `json:"future"`
	Data     string `json:"data"`
}

// nodes in Que definition
type node struct {
	value interface{}
	prev  *node
	next  *node
}

// Que definition
type LinkedQueue struct {
	head *node
	tail *node
	size int
}

// Selector query string constructor
type Selector struct {
	Members []SelectorMember `json:"$or"`
}

// Selector query string constructor
type SelectorMember struct {
	ID string `json:"ID"`
}

func Float32ToByte(float float32) []byte {
	bits := math.Float32bits(float)
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, bits)

	return bytes
}

func IntToBytes(n int) []byte {
	data := int64(n)
	bytebuf := bytes.NewBuffer([]byte{})
	binary.Write(bytebuf, binary.BigEndian, data)
	return bytebuf.Bytes()
}

func GenerateRecord(seed *rand.Rand, seq int) *Record {

	sha := sha256.New()
	sha.Write(IntToBytes(seq))
	// fmt.Println(hex.EncodeToString(sha.Sum(Float32ToByte(seed.Float32()))))
	record := Record{
		ID:       hex.EncodeToString(sha.Sum(nil)),
		Previous: "",
		Future:   "",
		Data:     fmt.Sprint(seq),
	}
	return &record
}

func RollDice(seed *rand.Rand, max int) int {
	return seed.Intn(max)
}
func RollDiceWithoutSeed(max int) int {
	return rand.Intn(max)
}

// func GetRandomListForAllTrees(treeNum int, recordNum int) []Record {

// }

func GetRandomTree(recordNum int) []*Record {

	var list []*Record

	rr := rand.New(rand.NewSource(time.Now().Unix()))
	// rr2 := rand.New(rand.NewSource(time.Now().Unix()))
	for i := 0; i < recordNum; i++ {
		newRecord := GenerateRecord(rr, i)
		if i != 0 {
			// previousNo := RollDice(rr2, i)
			previousNo := RollDiceWithoutSeed(i)
			// println("Dice = ", previousNo)
			newRecord.Previous = list[previousNo].ID

			// parse the future part of previous record
			list[previousNo].Future += "|" + newRecord.ID
		}

		list = append(list, newRecord)
	}

	// naiveJson, err := json.Marshal(list)
	// if err != nil {
	// }
	// fmt.Println(string(naiveJson))
	return list
}

func GetRandomTreeWithJump(recordNum int, jump int) []*Record {

	var list []*Record

	var previousNo int

	rr := rand.New(rand.NewSource(time.Now().Unix()))
	// rr2 := rand.New(rand.NewSource(time.Now().Unix()))
	for i := 0; i < recordNum; i++ {
		recordInicator := recordNum*jump + i

		newRecord := GenerateRecord(rr, recordInicator)
		if i > 0 {
			// previousNo := RollDice(rr2, i)
			if jump < straightTreeNum {
				previousNo = i - 1
			} else {
				previousNo = RollDiceWithoutSeed(i)
			}

			newRecord.Previous = list[previousNo].ID

			// parse the future part of previous record
			list[previousNo].Future += "|" + newRecord.ID
		}

		list = append(list, newRecord)
	}

	return list
}

func GetMultipleRandomTrees(recordNum int, treeNum int) []*Record {

	var list []*Record
	var nodeListForEachTree [][]*Record
	var treeNodeIndicator []int

	for i := 0; i < treeNum; i++ {
		tree := GetRandomTreeWithJump(recordNum, i)
		nodeListForEachTree = append(nodeListForEachTree, tree)
		treeNodeIndicator = append(treeNodeIndicator, 0)
	}
	rr := rand.New(rand.NewSource(time.Now().Unix()))
	for i := 0; i < treeNum*recordNum; i++ {
		dice := rr.Intn(treeNum)
		for treeNodeIndicator[dice] >= recordNum {
			dice = (dice + 1) % treeNum
		}
		list = append(list, nodeListForEachTree[dice][treeNodeIndicator[dice]])
		treeNodeIndicator[dice]++
	}
	// naiveJson, err := json.Marshal(list)
	// if err != nil {
	// }
	// fmt.Println(string(naiveJson))
	return list
}

// GetStringForSelctorMemberListFromString
func GetStringForSelctorMemberListFromString(future string) string {
	var memberList []SelectorMember
	strList := strings.Split(future, "|")
	if len(strList) == 0 {
		return ""
	}
	for i := 0; i < len(strList); i++ {
		// fmt.Println("i=", i, strList[i])
		TimeHold(len(strList))
		if strList[i] != "" {
			var member SelectorMember
			member.ID = strList[i]
			memberList = append(memberList, member)
		}
	}
	selector := Selector{
		Members: memberList,
	}
	q, err := json.Marshal(selector)
	if err != nil {
	}
	finalQstring := `{"selector":` + string(q) + `}`

	return finalQstring
}

// Get size
func (queue *LinkedQueue) Size() int {
	return queue.size
}

// Peek
func (queue *LinkedQueue) Peek() interface{} {
	if queue.head == nil {
		panic("Empty queue.")
	}
	return queue.head.value
}

// Add
func (queue *LinkedQueue) Add(value interface{}) {
	new_node := &node{value, queue.tail, nil}
	if queue.tail == nil {
		queue.head = new_node
		queue.tail = new_node
	} else {
		queue.tail.next = new_node
		queue.tail = new_node
	}
	queue.size++
	new_node = nil
}

// Remove
func (queue *LinkedQueue) Remove() {
	if queue.head == nil {
		panic("Empty queue.")
	}
	first_node := queue.head
	queue.head = first_node.next
	first_node.next = nil
	first_node.value = nil
	queue.size--
	first_node = nil
}

// InitLedger adds a base set of records to the ledger
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {

	records := GetMultipleRandomTrees(testNum, testTreeNum)

	for _, record := range records {
		recordJSON, err := json.Marshal(record)
		if err != nil {
			return err
		}

		err = ctx.GetStub().PutState(record.ID, recordJSON)
		if err != nil {
			return fmt.Errorf("failed to put to world state. %v", err)
		}
	}

	return nil
}

// CreateRecord issues a new record to the world state with given details.
func (s *SmartContract) CreateRecord(ctx contractapi.TransactionContextInterface, id string, previous string, future string, data string) error {
	exists, err := s.RecordExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("the record %s already exists", id)
	}

	record := Record{
		ID:       id,
		Previous: previous,
		Future:   future,
		Data:     data,
	}
	recordJSON, err := json.Marshal(record)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, recordJSON)
}

// ReadRecordbyData returns the record stored in the world state with given id.
func (s *SmartContract) ReadRecordbyData(ctx contractapi.TransactionContextInterface, data string) (*Record, error) {
	record, err := getQueryResultForQueryString(ctx, fmt.Sprintf(`{"selector":{"data":"%s"}}`, data))
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if record == nil {
		return nil, fmt.Errorf("the record %s does not exist", data)
	}

	return record[0], nil
}

// DeleteRecord deletes an given record from the world state.
func (s *SmartContract) DeleteRecord(ctx contractapi.TransactionContextInterface, id string) error {
	exists, err := s.RecordExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the record %s does not exist", id)
	}

	return ctx.GetStub().DelState(id)
}

// RecordExists returns true when record with given ID exists in world state
func (s *SmartContract) RecordExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	recordJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return recordJSON != nil, nil
}

// This is an example of a parameterized query where the query logic is baked into the chaincode,
// and accepting a single query parameter (owner).
// Only available on state databases that support rich query (e.g. CouchDB)
// Example: Parameterized rich query
func (s *SmartContract) QueryRecordsByPrevious(ctx contractapi.TransactionContextInterface, id string) ([]*Record, error) {
	queryString := fmt.Sprintf(`{"selector":{"Previous":"%s"}}`, id)
	return getQueryResultForQueryString(ctx, queryString)
}

// getQueryResultForQueryString executes the passed in query string.
// The result set is built and returned as a byte array containing the JSON results.
func getQueryResultForQueryString(ctx contractapi.TransactionContextInterface, queryString string) ([]*Record, error) {
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	return constructQueryResponseFromIterator(resultsIterator)
}

func constructQueryResponseFromIterator(resultsIterator shim.StateQueryIteratorInterface) ([]*Record, error) {
	var records []*Record
	for resultsIterator.HasNext() {
		queryResult, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		var record Record
		err = json.Unmarshal(queryResult.Value, &record)
		if err != nil {
			return nil, err
		}
		records = append(records, &record)
	}

	return records, nil
}


// AES componets
func PKCS5Padding(plaintext []byte, blockSize int) []byte{
    padding := blockSize-len(plaintext)%blockSize
    padtext := bytes.Repeat([]byte{byte(padding)},padding)
    return append(plaintext,padtext...)
}

//@brief:去除填充数据
func PKCS5UnPadding(origData []byte) []byte{
    length := len(origData)
    unpadding := int(origData[length-1])
    return origData[:(length - unpadding)]
}

//@brief:AES加密
func AesEncrypt(origData, key []byte) ([]byte, error){
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

	//AES分组长度为128位，所以blockSize=16，单位字节
    blockSize := block.BlockSize()
    origData = PKCS5Padding(origData,blockSize)					
    blockMode := cipher.NewCBCEncrypter(block,key[:blockSize])	//初始向量的长度必须等于块block的长度16字节
    crypted := make([]byte, len(origData))
    blockMode.CryptBlocks(crypted,origData)
    return crypted, nil
}

//@brief:AES解密
func AesDecrypt(crypted, key []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

	//AES分组长度为128位，所以blockSize=16，单位字节
    blockSize := block.BlockSize()
    blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])	//初始向量的长度必须等于块block的长度16字节
    origData := make([]byte, len(crypted))
    blockMode.CryptBlocks(origData, crypted)
    origData = PKCS5UnPadding(origData)
    return origData, nil
}


// Simulation for encryption and hash
func TimeHold(round int ) {
	var aeskey = []byte("12345678abcdefgh")
	// fmt.Println("Timehold Start.")
	for i := 0; i < round; i++ {
		// pass := []byte(RollDiceWithoutSeed(1000))
		pass := IntToBytes(RollDiceWithoutSeed(1000))
    	xpass, err := AesEncrypt(pass,aeskey)
		if err != nil {
			fmt.Println("what")
		}
		ciph := base64.StdEncoding.EncodeToString(xpass)
		sha := sha256.New()
		sha.Write([]byte(ciph))
		hex.EncodeToString(sha.Sum(nil))
	}
	// fmt.Println("Timehold Done.")
}



// Forward search
func (s *SmartContract) ForwardSearch(ctx contractapi.TransactionContextInterface, id string, seq int) ([]*Record, error) {

	// println("hello")
	var records []*Record
	recordJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if recordJSON == nil {
		// return records, nil
		return nil, fmt.Errorf("the record %s does not exist", id)
	}
	// println("passthefirst")

	var record Record
	err = json.Unmarshal(recordJSON, &record)

	records = append(records, &record)
	queryString := record.Future
	// fmt.Println(queryString)
	startTime := time.Now().UnixNano()
	for true {
		queryStringForSubmit := GetStringForSelctorMemberListFromString(queryString)
		// println("queryStringForSubmit is: ", queryStringForSubmit)
		resultsIterator, err1 := ctx.GetStub().GetQueryResult(queryStringForSubmit)
		if err1 != nil {
			return nil, err1
		}
		queryString = ""
		// println("daozhelema")
		for resultsIterator.HasNext() {
			// println("yes")
			queryResult, err2 := resultsIterator.Next()
			if err2 != nil {
				return nil, err2
			}
			// println("a,b")
			var record Record
			err2 = json.Unmarshal(queryResult.Value, &record)
			if err2 != nil {
				return nil, err2
			}
			records = append(records, &record)
			if record.Future != "" {
				queryString = queryString + "|" + record.Future
			}
		}
		defer resultsIterator.Close()
		// println("record string is: ", queryString)
		if queryString == "" {
			break
		}

	}
	endTime := time.Now().UnixNano()
	Ms := float64((endTime - startTime) / 1e6)
	fmt.Printf("Forward: Data %d completed in %f ms\n", seq, Ms)
	return records, nil
}

// Forward search with seq
func (s *SmartContract) ForwardSearchWithSeq(ctx contractapi.TransactionContextInterface) {
	testSeq := GetForwardTestSequence()
	for i := 0; i < len(testSeq); i++ {
		record, err := s.ReadRecordbyData(ctx, fmt.Sprint(testSeq[i]))
		if err != nil {
			fmt.Printf("No record with data %d", i)
			// return nil, err
		}

		s.ForwardSearch(ctx, record.ID, testSeq[i])

		if err != nil {
			fmt.Printf("No record with data %d", i)
		}

	}
	fmt.Sprintln("Finished!")
}

// Backward search with seq
func (s *SmartContract) BackwardSearchWithSeq(ctx contractapi.TransactionContextInterface) {
	testSeq := GetBackwardTestSequence()
	for i := 0; i < len(testSeq); i++ {
		record, err := s.ReadRecordbyData(ctx, fmt.Sprint(testSeq[i]))
		if err != nil {
			fmt.Sprintf("No record with data %d", i)
			// return nil, err
		}
		startTime := time.Now().UnixNano()
		s.BackwardSearch(ctx, record.ID)
		endTime := time.Now().UnixNano()
		Ms := float64((endTime - startTime) / 1e6)
		fmt.Printf("Backward: Data %d completed in %f ms\n", testSeq[i], Ms)

	}
	fmt.Sprintln("Finished!")
}

// Backward search from a given node in naive blockchain
func (s *SmartContract) BackwardSearch(ctx contractapi.TransactionContextInterface, id string) ([]*Record, error) {

	var records []*Record

	for true {
		recordJSON, err := ctx.GetStub().GetState(id)
		if err != nil {
			return nil, fmt.Errorf("failed to read from world state: %v", err)
		}
		if recordJSON == nil {
			return records, nil
			// return nil, fmt.Errorf("the record %s does not exist", id)
		}

		var record Record
		err = json.Unmarshal(recordJSON, &record)

		if err != nil {
			return nil, err
		}
		// println("hello\nunmarshal id = %s", record.ID)

		records = append(records, &record)
		id = record.Previous
		// println("id = %s", id)
	}

	return records, nil
}

// AutoTest with both direction
func (s *SmartContract) AutoTest(ctx contractapi.TransactionContextInterface) {
	s.ForwardSearchWithSeq(ctx)
	s.BackwardSearchWithSeq(ctx)
}

// GetAllRecords returns all records found in world state
func (s *SmartContract) GetAllRecords(ctx contractapi.TransactionContextInterface) ([]*Record, error) {
	// range query with empty string for startKey and endKey does an
	// open-ended query of all records in the chaincode namespace.
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var records []*Record
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var record Record
		err = json.Unmarshal(queryResponse.Value, &record)
		if err != nil {
			return nil, err
		}
		records = append(records, &record)
	}

	return records, nil
}

func main() {
	recordChaincode, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		log.Panicf("Error creating niaiveTree-basic chaincode: %v", err)
	}

	if err := recordChaincode.Start(); err != nil {
		log.Panicf("Error starting niaiveTree-basic chaincode: %v", err)
	}
}
