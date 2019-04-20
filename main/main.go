package main

import (
	"flag"
	"fmt"
	// "github.com/mattsteencpp/go-daily-tracker/tracker"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"time"
)

const timeFormat = "3:04pm"

type TrackerData struct {
	Time string `yaml:"time"`
	Entries []TrackerEntry `yaml:"entries"`
	Todos []TrackerTodo `yaml:"todos"`
}

type TrackerEntry struct {
	Name string `yaml:"name"`
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
	delta *= 0.25
	if action == "s" {
		delta *= -1
	}

	idx := letterToIdx(letter)
	data.Entries[idx].Total += delta

	updateTime(data, delta)
}

func getLogTime(data *TrackerData) time.Time {
	timeKey := data.Time
	logTime, err := time.Parse(timeFormat, timeKey)
	if err != nil {
		fmt.Printf("Failed to parse time: %v\n", err)
		os.Exit(1)
	}
	return logTime
}

func smartUpdateValue(data *TrackerData, letter string) {
	// add time to the selected entry until caught up to current time
	logTime := getLogTime(data)

	// get the current datetime and convert to time; if we try to subtract
	// directly, it assumes that logTime is on day 1 of the epoch
	currentDateTime := time.Now().Format(timeFormat)
	currentTime, err := time.Parse(timeFormat, currentDateTime)
	if err != nil {
		fmt.Printf("Failed to parse time: %v\n", err)
		os.Exit(1)
	}

	timeBlocks := currentTime.Sub(logTime).Round(15 * time.Minute)
	delta := timeBlocks.Minutes() / 60.0

	idx := letterToIdx(letter)
	data.Entries[idx].Total += delta

	updateTime(data, delta)
}

func updateTime(data *TrackerData, delta float64) {
	logTime := getLogTime(data)
	logTime = logTime.Add(time.Minute * time.Duration(int(60 * delta)))
	newTime := logTime.Format(timeFormat)
	data.Time = newTime
}

func setTime(data *TrackerData, newTime string) {
	_, err := time.Parse(timeFormat, newTime)
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

func printState(data TrackerData) {
	entries := data.Entries
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		letter := idxToLetter(i)
		fmt.Printf("%s) %s: %vh\n", letter, entry.Name, entry.Total)
	}
	logTime := data.Time
	fmt.Printf(logTime)
	fmt.Printf("\n\n")

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
	data.Entries = append(data.Entries[:idx], data.Entries[idx + 1:]...)
}

func createTodo(data *TrackerData, todo string) {
	newTodo := TrackerTodo{Content: todo}
	data.Todos = append(data.Todos, newTodo)
}

func deleteTodo(data *TrackerData, letter string) {
	idx := letterToIdx(letter)
	data.Todos = append(data.Todos[:idx], data.Todos[idx + 1:]...)
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
	if action == "h" {
		fmt.Println("dt by Matt Steen")
		fmt.Println("v1.0.0")
		fmt.Println("Usage: ")
		fmt.Println("'dt' to get current status")
		fmt.Println("'dt a a' to add 15 minutes to entry a")
		fmt.Println("'dt a 3b' to add 45 minutes to entry b")
		fmt.Println("'dt s 2b' to subtract 30 minutes from entry b")
		fmt.Println("'dt t 8:00am' to set today's start time to 8am")
		fmt.Println("'dt n bounce_backs' to add a new entry called bounce_backs")
		fmt.Println("'dt m d loyalty' to rename entry d to loyalty")
		fmt.Println("'dt d e' to delete entry e")
		fmt.Println("'dt r' to reset all entries to 0 and the start time to the previous day")
		fmt.Println("'dt todo \"review Corey's PR\"' to add a new todo")
		fmt.Println("'dt tm d loyalty' to rename todo d to loyalty")
		fmt.Println("'dt tr a b' to swap todos a and b")
		fmt.Println("'dt c a' to checkoff todo a")
	} else if action == "a" || action == "s" { // add or subtract
		input := flag.Arg(1)
		updateValue(&trackerData, action, input)
	} else if action == "u" { // smart update
		letter := flag.Arg(1)
		smartUpdateValue(&trackerData, letter)
	} else if action == "t" { // set start time for the day
		newTime := flag.Arg(1)
		setTime(&trackerData, newTime)
	} else if action == "r" { // reset all entries
		resetEntries(&trackerData)
	} else if action == "n" { // new entry
		name := flag.Arg(1)
		createEntry(&trackerData, name)
	} else if action == "m" { // mv entry
		letter := flag.Arg(1)
		newName := flag.Arg(2)
		renameEntry(&trackerData, letter, newName)
	} else if action == "d" { // delete entry
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
	} else if action == "c" { // check off a todo
		letter := flag.Arg(1)
		deleteTodo(&trackerData, letter)
	}
	if action != "h" {
		printState(trackerData)
	}

	output, err := yaml.Marshal(trackerData)
	err = ioutil.WriteFile(fullpath, output, 0666)
	if err != nil {
		fmt.Printf("There was an error saving changes to file\n")
	}
}
