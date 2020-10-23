package main

import (
	"flag"
	"fmt"
	"github.com/gookit/color"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"time"
)

type trackerData struct {
	Time       string         `yaml:"time"`
	TimeFormat string         `yaml:"format"`
	BlockSize  int64          `yaml:"block_size"`
	Entries    []trackerEntry `yaml:"entries"`
	Todos      []trackerTodo  `yaml:"todos"`
}

type trackerEntry struct {
	Name  string  `yaml:"name"`
	Total float64 `yaml:"total"`
}

type trackerTodo struct {
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

func updateValue(data *trackerData, action, input string) {
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

func getLogTime(data *trackerData) time.Time {
	timeKey := data.Time
	logTime, err := time.Parse(data.TimeFormat, timeKey)
	if err != nil {
		fmt.Printf("Failed to parse time: %v\n", err)
		os.Exit(1)
	}
	return logTime
}

func setBlockSize(data *trackerData, size string) {
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

func smartUpdateTime(data *trackerData) {
	currentDateTime := time.Now().Round(time.Duration(data.BlockSize) * time.Minute)
	data.Time = currentDateTime.Format(data.TimeFormat)
}

func smartUpdateValue(data *trackerData, letter string) {
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

func updateTime(data *trackerData, delta float64) {
	logTime := getLogTime(data)
	logTime = logTime.Add(time.Minute * time.Duration(delta))
	newTime := logTime.Format(data.TimeFormat)
	data.Time = newTime
}

func setTime(data *trackerData, newTime string) {
	_, err := time.Parse(data.TimeFormat, newTime)
	if err != nil {
		fmt.Printf("Failed to parse time: %v\n", err)
		os.Exit(1)
	}
	data.Time = newTime
}

func resetEntries(data *trackerData) {
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

func printState(data trackerData, summaryOnly, showTodos bool) {
	entries := data.Entries
	totalTime := 0.0
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if summaryOnly && entry.Total == 0 {
			continue
		}

		letter := idxToLetter(i)
		if entry.Name != "lunch" && entry.Name != "me time" {
			totalTime += entry.Total
		}

		formattedDuration := formatDuration(entry.Total)
		fmt.Printf("%s) %s: %vh\n", letter, entry.Name, formattedDuration)
	}
	color.Style{color.FgCyan, color.OpBold}.Printf("Total: %vh\n", formatDuration(totalTime))
	color.Style{color.FgCyan, color.OpBold}.Printf("Logged Time: %v\n", data.Time)
	currentDateTime := time.Now()
	color.Style{color.FgCyan, color.OpBold}.Printf("Current Time: %v\n\n", currentDateTime.Format(data.TimeFormat))

	if !showTodos {
		return
	}

	fmt.Printf("TODO:\n")
	for idx := 0; idx < len(data.Todos); idx++ {
		letter := idxToLetter(idx)
		fmt.Printf("%s) %s\n", letter, data.Todos[idx].Content)
	}
	fmt.Println("")
}

func createEntry(data *trackerData, name string) {
	newEntry := trackerEntry{Name: name, Total: 0.0}
	data.Entries = append(data.Entries, newEntry)
}

func renameEntry(data *trackerData, letter string, newName string) {
	idx := letterToIdx(letter)
	data.Entries[idx].Name = newName
}

func deleteEntry(data *trackerData, letter string) {
	idx := letterToIdx(letter)
	data.Entries = append(data.Entries[:idx], data.Entries[idx+1:]...)
}

func createTodo(data *trackerData, todo string) {
	newTodo := trackerTodo{Content: todo}
	data.Todos = append(data.Todos, newTodo)
}

func deleteTodo(data *trackerData, letter string) {
	idx := letterToIdx(letter)
	data.Todos = append(data.Todos[:idx], data.Todos[idx+1:]...)
}

func renameTodo(data *trackerData, letter string, newName string) {
	idx := letterToIdx(letter)
	data.Todos[idx].Content = newName
}

func swapTodos(data *trackerData, letterOne string, letterTwo string) {
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
	fmt.Println("'dt' to get a summary of current status")
	fmt.Println("'dt add a' to add a block to entry a")
	fmt.Println("'dt add 3b' to add 3 blocks to entry b")
	fmt.Println("'dt subtract 2b' to subtract 2 blocks from entry b")
	fmt.Println("'dt blocksize 10' to set the block size going forward to 10 minutes")
	fmt.Println("'dt time 8:00am' to set today's start time to 8am")
	fmt.Println("'dt start' to set today's start time to the current time")
	fmt.Println("'dt new \"project 1\"' to add a new entry called 'project 1'")
	fmt.Println("'dt mv d loyalty' to rename entry d to loyalty")
	fmt.Println("'dt delete e' to delete entry e")
	fmt.Println("'dt reset' to reset all entries to 0 and the start time to the previous day")
	fmt.Println("'dt todo \"review Corey's PR\"' to add a new todo")
	fmt.Println("'dt tm d loyalty' to rename todo d to loyalty")
	fmt.Println("'dt tr a b' to swap todos a and b")
	fmt.Println("'dt checkoff a' to check off todo a")
	fmt.Println("'dt summary' to show a summary for the day")
	fmt.Println("'dt all' to show all entries for the day")
}

func main() {
	fullpath := "/home/msteen/.daily-tracker.yaml"
	body, err := ioutil.ReadFile(fullpath)
	if err != nil {
		fmt.Printf("There was an error reading the yaml file\n")
		os.Exit(1)
	}

	var data trackerData
	yaml.Unmarshal(body, &data)

	summaryOnly := true
	showTodos := false

	flag.Parse()
	action := flag.Arg(0)
	if action == "h" || action == "help" {
		printHelp()
	} else if action == "a" || action == "s" || action == "add" || action == "subtract" { // add or subtract
		input := flag.Arg(1)
		updateValue(&data, action, input)
	} else if action == "u" || action == "update" { // smart update
		letter := flag.Arg(1)
		smartUpdateValue(&data, letter)
	} else if action == "t" || action == "time" { // set start time for the day
		newTime := flag.Arg(1)
		setTime(&data, newTime)
	} else if action == "b" || action == "blocksize" { // set block size for future updates
		size := flag.Arg(1)
		setBlockSize(&data, size)
	} else if action == "st" || action == "start" { // set start time to the current time
		smartUpdateTime(&data)
	} else if action == "r" || action == "reset" { // reset all entries
		resetEntries(&data)
	} else if action == "n" || action == "new" { // new entry
		name := flag.Arg(1)
		createEntry(&data, name)
	} else if action == "m" || action == "mv" { // mv entry
		letter := flag.Arg(1)
		newName := flag.Arg(2)
		renameEntry(&data, letter, newName)
	} else if action == "d" || action == "delete" { // delete entry
		letter := flag.Arg(1)
		deleteEntry(&data, letter)
	} else if action == "todo" { // add a todo
		showTodos = true
		todo := flag.Arg(1)
		createTodo(&data, todo)
	} else if action == "tm" { // reword a todo
		showTodos = true
		letter := flag.Arg(1)
		newName := flag.Arg(2)
		renameTodo(&data, letter, newName)
	} else if action == "tr" { // reorder todos
		showTodos = true
		letterOne := flag.Arg(1)
		letterTwo := flag.Arg(2)
		swapTodos(&data, letterOne, letterTwo)
	} else if action == "c" || action == "checkoff" { // check off a todo
		showTodos = true
		letter := flag.Arg(1)
		deleteTodo(&data, letter)
	} else if action == "sum" || action == "summary" { // display a summary
		summaryOnly = true
		showTodos = false
	} else if action == "all" || action == "" { // display all entries, not just those with data
		summaryOnly = false
		showTodos = true
	}

	if action != "h" && action != "help" {
		printState(data, summaryOnly, showTodos)
	}

	output, err := yaml.Marshal(data)
	err = ioutil.WriteFile(fullpath, output, 0666)
	if err != nil {
		fmt.Printf("There was an error saving changes to file\n")
	}
}
