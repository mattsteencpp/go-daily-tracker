package main

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"time"
)

type TrackerData struct {
	Time       string         `yaml:"time"`
	TimeFormat string         `yaml:"format"`
	BlockSize  int64          `yaml:"block_size"`
	Entries    []TrackerEntry `yaml:"entries"`
	Todos      []TrackerTodo  `yaml:"todos"`
}

type TrackerEntry struct {
	Name  string  `yaml:"name"`
	Total float64 `yaml:"total"`
}

type TrackerTodo struct {
	Content string `yaml:"content"`
}

func idxToLetter(idx int) string {
	letter := string(idx + 97)
	return letter
}

func letterToIdx(letter string) int {
	idx := int(letter[0]) - 97
	return idx
}

func updateValue(data *TrackerData, action, input string) {
	// action should be a for add or s for subtract
	// input should be the letter of the entry to update,
	// optionally preceded by an integer multiplier
	regex := *regexp.MustCompile("([0-9]*)([a-z]+)")
	result := regex.FindAllStringSubmatch(input, 1)
	rawQuantity := result[0][1]
	if rawQuantity == "" {
		rawQuantity = "1"
	}
	letter := result[0][2]
	delta, err := strconv.ParseFloat(rawQuantity, 64)
	if err != nil {
		fmt.Printf("Failed to parse multiplier: %v\n", err)
		os.Exit(1)
	}
	delta *= float64(data.BlockSize)
	if action == "s" || action == "subtract" {
		delta *= -1
	}

	idx := letterToIdx(letter)
	data.Entries[idx].Total += delta

	// don't allow negative totals
	if data.Entries[idx].Total < 0 {
		delta -= data.Entries[idx].Total
		data.Entries[idx].Total = 0.0
		fmt.Printf("Negative totals are not permitted\n\n")
		os.Exit(1)
	}

	updateTime(data, delta)
}

func getLogTime(data *TrackerData) time.Time {
	timeKey := data.Time
	logTime, err := time.Parse(data.TimeFormat, timeKey)
	if err != nil {
		fmt.Printf("Failed to parse time: %v\n", err)
		os.Exit(1)
	}
	return logTime
}

func setBlockSize(data *TrackerData, size string) {
	blockSize, err := strconv.ParseInt(size, 10, 64)
	if err != nil {
		fmt.Printf("Failed to parse block size (must be an integer): %v\n", err)
		os.Exit(1)
	}
	if blockSize < 1 || blockSize > 60 {
		fmt.Printf("Block size must be between 1 and 60 minutes: %v\n", err)
		os.Exit(1)
	}
	for i := 0; i < len(data.Entries); i++ {
		if data.Entries[i].Total > 0 {
			fmt.Printf("Block size cannot be updated after logging time for the day (total must be 0)\n")
			os.Exit(1)
		}
	}
	data.BlockSize = blockSize
}

func smartUpdateTime(data *TrackerData) {
	currentDateTime := time.Now().Round(time.Duration(data.BlockSize) * time.Minute)
	data.Time = currentDateTime.Format(data.TimeFormat)
}

func smartUpdateValue(data *TrackerData, letter string) {
	// add time to the selected entry until caught up to current time
	logTime := getLogTime(data)

	// get the current datetime and convert to time; if we try to subtract
	// directly, it assumes that logTime is on day 1 of the epoch
	currentDateTime := time.Now().Format(data.TimeFormat)
	currentTime, err := time.Parse(data.TimeFormat, currentDateTime)
	if err != nil {
		fmt.Printf("Failed to parse time: %v\n", err)
		os.Exit(1)
	}

	timeBlocks := currentTime.Sub(logTime).Round(time.Duration(data.BlockSize) * time.Minute)
	delta := timeBlocks.Minutes()

	idx := letterToIdx(letter)
	data.Entries[idx].Total += delta

	updateTime(data, delta)
}

func updateTime(data *TrackerData, delta float64) {
	logTime := getLogTime(data)
	logTime = logTime.Add(time.Minute * time.Duration(delta))
	newTime := logTime.Format(data.TimeFormat)
	data.Time = newTime
}

func setTime(data *TrackerData, newTime string) {
	_, err := time.Parse(data.TimeFormat, newTime)
	if err != nil {
		fmt.Printf("Failed to parse time: %v\n", err)
		os.Exit(1)
	}
	data.Time = newTime
}

func resetEntries(data *TrackerData) {
	totalTime := 0.0
	for i := 0; i < len(data.Entries); i++ {
		totalTime -= data.Entries[i].Total
		data.Entries[i].Total = 0
	}
	updateTime(data, totalTime)
}

func formatDuration(totalMinutes float64) string {
	if totalMinutes == 0.0 {
		return "0"
	}
	duration := time.Duration(totalMinutes) * time.Minute
	hours := duration / time.Hour
	duration -= hours * time.Hour
	minutes := duration / time.Minute
	return fmt.Sprintf("%d:%02d", hours, minutes)
}

func printState(data TrackerData) {
	entries := data.Entries
	totalTime := 0.0
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		letter := idxToLetter(i)
		if entry.Name != "lunch" && entry.Name != "me time" {
			totalTime += entry.Total
		}

		formattedDuration := formatDuration(entry.Total)
		fmt.Printf("%s) %s: %vh\n", letter, entry.Name, formattedDuration)
	}
	fmt.Printf("Total: %vh\n", formatDuration(totalTime))
	fmt.Printf("Logged Time: %v\n", data.Time)
	currentDateTime := time.Now()
	fmt.Printf("Current Time: %v\n\n", currentDateTime.Format(data.TimeFormat))

	fmt.Printf("TODO:\n")
	for idx := 0; idx < len(data.Todos); idx++ {
		letter := idxToLetter(idx)
		fmt.Printf("%s) %s\n", letter, data.Todos[idx].Content)
	}
	fmt.Println("")
}

func createEntry(data *TrackerData, name string) {
	newEntry := TrackerEntry{Name: name, Total: 0.0}
	data.Entries = append(data.Entries, newEntry)
}

func renameEntry(data *TrackerData, letter string, newName string) {
	idx := letterToIdx(letter)
	data.Entries[idx].Name = newName
}

func deleteEntry(data *TrackerData, letter string) {
	idx := letterToIdx(letter)
	data.Entries = append(data.Entries[:idx], data.Entries[idx+1:]...)
}

func createTodo(data *TrackerData, todo string) {
	newTodo := TrackerTodo{Content: todo}
	data.Todos = append(data.Todos, newTodo)
}

func deleteTodo(data *TrackerData, letter string) {
	idx := letterToIdx(letter)
	data.Todos = append(data.Todos[:idx], data.Todos[idx+1:]...)
}

func renameTodo(data *TrackerData, letter string, newName string) {
	idx := letterToIdx(letter)
	data.Todos[idx].Content = newName
}

func swapTodos(data *TrackerData, letterOne string, letterTwo string) {
	idxOne := letterToIdx(letterOne)
	idxTwo := letterToIdx(letterTwo)
	tempContent := data.Todos[idxTwo].Content
	data.Todos[idxTwo].Content = data.Todos[idxOne].Content
	data.Todos[idxOne].Content = tempContent
}

func printHelp() {
	fmt.Println("dt by Matt Steen")
	fmt.Println("v1.0.0")
	fmt.Println("Usage: ")
	fmt.Println("'dt' to get current status")
	fmt.Println("'dt add a' to add a block to entry a")
	fmt.Println("'dt add 3b' to add 3 blocks to entry b")
	fmt.Println("'dt subtract 2b' to subtract 2 blocks from entry b")
	fmt.Println("'dt blocksize 10' to set the block size going forward to 10 minutes")
	fmt.Println("'dt time 8:00am' to set today's start time to 8am")
	fmt.Println("'dt start' to set today's start time to the current time")
	fmt.Println("'dt new \"project 1\" to add a new entry called 'project 1'")
	fmt.Println("'dt mv d loyalty' to rename entry d to loyalty")
	fmt.Println("'dt delete e' to delete entry e")
	fmt.Println("'dt reset' to reset all entries to 0 and the start time to the previous day")
	fmt.Println("'dt todo \"review Corey's PR\"' to add a new todo")
	fmt.Println("'dt tm d loyalty' to rename todo d to loyalty")
	fmt.Println("'dt tr a b' to swap todos a and b")
	fmt.Println("'dt checkoff a' to check off todo a")
}

func main() {
	fullpath := "/home/msteen/.daily-tracker.yaml"
	body, err := ioutil.ReadFile(fullpath)
	if err != nil {
		fmt.Printf("There was an error reading the yaml file\n")
		os.Exit(1)
	}

	var trackerData TrackerData
	yaml.Unmarshal(body, &trackerData)

	flag.Parse()
	action := flag.Arg(0)
	if action == "h" || action == "help" {
		printHelp()
	} else if action == "a" || action == "s" || action == "add" || action == "subtract" { // add or subtract
		input := flag.Arg(1)
		updateValue(&trackerData, action, input)
	} else if action == "u" || action == "update" { // smart update
		letter := flag.Arg(1)
		smartUpdateValue(&trackerData, letter)
	} else if action == "t" || action == "time" { // set start time for the day
		newTime := flag.Arg(1)
		setTime(&trackerData, newTime)
	} else if action == "b" || action == "blocksize" { // set block size for future updates
		size := flag.Arg(1)
		setBlockSize(&trackerData, size)
	} else if action == "st" || action == "start" { // set start time to the current time
		smartUpdateTime(&trackerData)
	} else if action == "r" || action == "reset" { // reset all entries
		resetEntries(&trackerData)
	} else if action == "n" || action == "new" { // new entry
		name := flag.Arg(1)
		createEntry(&trackerData, name)
	} else if action == "m" || action == "mv" { // mv entry
		letter := flag.Arg(1)
		newName := flag.Arg(2)
		renameEntry(&trackerData, letter, newName)
	} else if action == "d" || action == "delete" { // delete entry
		letter := flag.Arg(1)
		deleteEntry(&trackerData, letter)
	} else if action == "todo" { // add a todo
		todo := flag.Arg(1)
		createTodo(&trackerData, todo)
	} else if action == "tm" { // reword a todo
		letter := flag.Arg(1)
		newName := flag.Arg(2)
		renameTodo(&trackerData, letter, newName)
	} else if action == "tr" { // reorder todos
		letterOne := flag.Arg(1)
		letterTwo := flag.Arg(2)
		swapTodos(&trackerData, letterOne, letterTwo)
	} else if action == "c" || action == "checkoff" { // check off a todo
		letter := flag.Arg(1)
		deleteTodo(&trackerData, letter)
	}
	if action != "h" && action != "help" {
		printState(trackerData)
	}

	output, err := yaml.Marshal(trackerData)
	err = ioutil.WriteFile(fullpath, output, 0666)
	if err != nil {
		fmt.Printf("There was an error saving changes to file\n")
	}
}
