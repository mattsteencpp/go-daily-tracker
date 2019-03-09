package main

import (
	"flag"
    "fmt"
    // "github.com/mattsteencpp/go-daily-tracker/tracker"
	"gopkg.in/ini.v1"
	"os"
	"regexp"
	"strconv"
	"time"
)

const timeFormat = "3:04pm"

func idxToLetter(idx int) string {
	letter := string(idx + 97)
	return letter
}

func letterToIdx(letter string) int {
	idx := int(letter[0]) - 97
	return idx
}

func updateValue(data *ini.File, action, input string) {
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
	keys := data.Section("entries").Keys()
	key := keys[idx]
	value, _ := key.Float64()
	value += delta
	strValue := fmt.Sprintf("%.2f", value)
	key.SetValue(strValue)

	updateTime(data, delta)
}

func updateTime(data *ini.File, delta float64) {
	// update log time
	timeKey := data.Section("").Key("time")
	logTime, err := timeKey.TimeFormat(timeFormat)
    if err != nil {
        fmt.Printf("Failed to parse time: %v\n", err)
        os.Exit(1)
    }
	logTime = logTime.Add(time.Minute * time.Duration(int(60 * delta)))
	newTime := logTime.Format(timeFormat)
	timeKey.SetValue(newTime)
}

func setTime(data *ini.File, newTime string) {
	timeKey := data.Section("").Key("time")
	_, err := time.Parse(timeFormat, newTime)
    if err != nil {
        fmt.Printf("Failed to parse time: %v\n", err)
        os.Exit(1)
    }
	timeKey.SetValue(newTime)
}

func resetEntries(data *ini.File) {
	keys := data.Section("entries").Keys()
	totalTime := 0.0
	for i := 0; i < len(keys); i++ {
		key := keys[i]
		value, _ := key.Float64()
		totalTime -= value
		key.SetValue("0")
	}
	updateTime(data, totalTime)
}

func printState(data *ini.File) {
	keys := data.Section("entries").Keys()
	for i := 0; i < len(keys); i++ {
		key := keys[i]
		value, _ := key.Float64()
		letter := idxToLetter(i)
		fmt.Printf("%s) %s: %vh\n", letter, key.Name(), value)
	}
	logTime := data.Section("").Key("time").String()
	fmt.Printf(logTime)
	fmt.Println("\n")
	// TODO: print daily todos from file
}

func createEntry(data *ini.File, name string) {
	data.Section("entries").NewKey(name, "0")
}

func renameEntry(data *ini.File, letter string, newName string) {
	idx := letterToIdx(letter)
	keys := data.Section("entries").Keys()
	key := keys[idx]
	value := key.String()
	data.Section("entries").DeleteKey(key.Name())
	data.Section("entries").NewKey(newName, value)
	// TODO: handle ordering
}

func deleteEntry(data *ini.File, letter string) {
	idx := letterToIdx(letter)
	keys := data.Section("entries").Keys()
	key := keys[idx]
	data.Section("entries").DeleteKey(key.Name())
}

func main() {
	fullpath := "/home/msteen/.daily-tracker.ini"
	data, err := ini.Load(fullpath)
    if err != nil {
        fmt.Printf("Failed to read file: %v'\n", err)
        os.Exit(1)
    }
	flag.Parse()
	action := flag.Arg(0)
	if action == "h" {
		fmt.Println("daily-tracker by Matt Steen")
		fmt.Println("v1.0.0")
		fmt.Println("Usage: ")
		fmt.Println("'daily-tracker' to get current status")
		fmt.Println("'daily-tracker a a' to add 15 minutes to entry a")
		fmt.Println("'daily-tracker a 3b' to add 45 minutes to entry b")
		fmt.Println("'daily-tracker s 2b' to subtract 30 minutes from entry b")
		fmt.Println("'daily-tracker t 8:00am' to set today's start time to 8am")
		fmt.Println("'daily-tracker n bounce_backs' to add a new entry called bounce_backs")
		fmt.Println("'daily-tracker m d loyalty' to rename entry d to loyalty")
		fmt.Println("'daily-tracker d e' to delete entry e")
		fmt.Println("'daily-tracker r' to reset all entries to 0 and the start time to the previous day")
		fmt.Println("'daily-tracker todo \"review Corey's PR\"' to add a new todo")
		// fmt.Println("'daily-tracker c 1' to reorder todo 1")
		fmt.Println("'daily-tracker c 1' to checkoff todo 1")
	} else if action == "a" || action == "s" { // add or subtract
		input := flag.Arg(1)
		updateValue(data, action, input)
	} else if action == "t" { // set start time for the day
		newTime := flag.Arg(1)
		setTime(data, newTime)
	} else if action == "r" { // reset all entries
		resetEntries(data)
	} else if action == "n" { // new entry
		name := flag.Arg(1)
		createEntry(data, name)
	} else if action == "m" { // mv entry
		letter := flag.Arg(1)
		newName := flag.Arg(2)
		renameEntry(data, letter, newName)
	} else if action == "d" { // delete entry
		letter := flag.Arg(1)
		deleteEntry(data, letter)
	} else if action == "todo" { // add a todo
		// TODO: implement this
	} else if action == "tr" { // reorder todos
		// TODO: implement this; think through how it should work
	} else if action == "c" { // check off a todo
		// TODO: implement this
	}
	printState(data)
	data.SaveTo(fullpath)
}
