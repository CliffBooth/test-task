package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	incomingClientCame      = 1
	incomingClientTookTable = 2
	incomingClientWaiting   = 3
	incomingClientLeft      = 4

	outcomingClientLeft      = 11
	outcomingClientTookTable = 12
	outcomingError           = 13

	notOpenMsg         = "NotOpenYet"
	alreadyPresentMsg  = "YouShallNotPass"
	placeIsBusyMsg     = "PlaceIsBusy"
	clientUnknownMsg   = "ClientUnknown"
	canWaitNoLongerMsg = "ICanWaitNoLonger!"
)

var (
	IncomingEventIDs = []int{
		incomingClientCame,
		incomingClientTookTable,
		incomingClientWaiting,
		incomingClientLeft,
	}
)

type ComputerClub struct {
	TablesNumber int
	StartTime    time.Time
	EndTime      time.Time
	Tariff       int

	TablesTaken int

	Clients      map[string]*ClientInfo
	WaitingQueue []string

	TableStats []TableStat
}

type ClientInfo struct {
	AtTable bool      // сел ли клиент за стол
	Table   int       // номер стола
	Time    time.Time // время, когда клиент сел за стол
}

type TableStat struct {
	TimeSpent time.Time
	Revenue   int
}

type Event struct {
	T          time.Time
	ClientName string
	ID         int

	Table int
}

func (e Event) String() string {
	if e.ID == incomingClientTookTable {
		return fmt.Sprintf("%02d:%02d %d %s %d", e.T.Hour(), e.T.Minute(), e.ID, e.ClientName, e.Table)
	} else {
		return fmt.Sprintf("%02d:%02d %d %s", e.T.Hour(), e.T.Minute(), e.ID, e.ClientName)
	}
}

func main() {
	args := os.Args
	if len(args) < 2 {
		panic("please provide a file path as first argument")
	}
	filePath := args[1]

	Run(filePath, os.Stdout)
}

// Run принимает на вход путь до файла, который надо считать, и записывает вывод в out.
func Run(filePath string, out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		panic("error opening file: " + err.Error())
	}

	club := ComputerClub{
		Clients: make(map[string]*ClientInfo),
	}

	scanner := bufio.NewScanner(file)

	club.TablesNumber, err = readNextPositiveInt(scanner)
	if err != nil {
		fmt.Fprint(out, err.Error())
		return
	}
	club.TableStats = make([]TableStat, club.TablesNumber)

	club.StartTime, club.EndTime, err = readNextTime(scanner)
	if err != nil {
		fmt.Fprint(out, err.Error())
		return
	}

	club.Tariff, err = readNextPositiveInt(scanner)
	if err != nil {
		fmt.Fprint(out, err.Error())
		return
	}

	// сначала парсим все строки и получаем список событий
	events := []Event{}
	for scanner.Scan() {
		line := scanner.Text()

		var prevTime *time.Time = nil
		if len(events) != 0 {
			prevTime = &events[len(events)-1].T
		}

		event, err := club.parseEvent(line, prevTime)
		// при неправильном формате входных данных выводим строку с ошибкой
		if err != nil {
			fmt.Fprint(out, line)
			return
		}
		events = append(events, event)
	}

	err = file.Close()
	if err != nil {
		panic(err)
	}

	// обрабатываем все события
	fmt.Fprintf(out, "%02d:%02d\n", club.StartTime.Hour(), club.StartTime.Minute())
	for _, event := range events {
		fmt.Fprintln(out, event)
		message := club.processEvent(event)
		if message != "" {
			fmt.Fprintln(out, message)
		}
	}

	// клиенты уходят
	clientList := []string{}
	for client := range club.Clients {
		clientList = append(clientList, client)
	}
	sort.Strings(clientList)
	for _, client := range clientList {
		fmt.Fprintln(out, getMessage(club.EndTime, outcomingClientLeft, client))
		club.clientLeavesTable(client, club.EndTime)
	}
	fmt.Fprintf(out, "%02d:%02d\n", club.EndTime.Hour(), club.EndTime.Minute())

	// в конце выводим статистику по столам
	for i, stat := range club.TableStats {
		fmt.Fprintf(out, "%d %d %02d:%02d", i+1, stat.Revenue, stat.TimeSpent.Hour(), stat.TimeSpent.Minute())
		if i < len(club.TableStats)-1 {
			fmt.Fprintf(out, "\n")
		}
	}
}

func (c *ComputerClub) parseEvent(line string, prevTime *time.Time) (Event, error) {
	params := strings.Fields(line)
	if len(params) < 3 || len(params) > 4 {
		return Event{}, errors.New("invalid line")
	}

	regex := regexp.MustCompile(`^\d{2}:\d{2}$`)
	if !regex.MatchString(params[0]) {
		return Event{}, errors.New("invalid time")
	}
	time, err := time.Parse(time.TimeOnly, params[0]+":00")
	if err != nil {
		return Event{}, err
	}

	// проверяем, что событие произошло дальше по времени
	if prevTime != nil && time.Before(*prevTime) {
		return Event{}, errors.New("invalid time")
	}

	// проверяем корректность id события
	id, err := strconv.Atoi(params[1])
	if err != nil {
		return Event{}, err
	}
	contains, _ := Contains(IncomingEventIDs, id)
	if !contains {
		return Event{}, errors.New("invalid event id")
	}

	clientName := params[2]
	regex = regexp.MustCompile(`^[\d\w_-]+$`)
	if !regex.MatchString(clientName) {
		return Event{}, errors.New("invalid client name")
	}

	switch id {
	case incomingClientTookTable:
		if len(params) != 4 {
			return Event{}, errors.New("invalid line")
		}
		// проверяем, что стол коррентный
		tableNumber, err := strconv.Atoi(params[3])
		if err != nil {
			return Event{}, err
		}
		if tableNumber < 1 || tableNumber > c.TablesNumber {
			return Event{}, errors.New("invalid table")
		}

		return Event{
			T:          time,
			ClientName: clientName,
			ID:         id,
			Table:      tableNumber,
		}, nil

	default:
		return Event{
			T:          time,
			ClientName: clientName,
			ID:         id,
		}, nil
	}
}

// readNextPositiveInt читает следующую строку из сканнера, и пытается конвертировать в int.
// Если не получается, возвращает ошибку со считанной строкой.
func readNextPositiveInt(scanner *bufio.Scanner) (int, error) {
	end := scanner.Scan()
	if !end {
		return 0, errors.New("not enough lines in file")
	}
	tablesString := scanner.Text()
	value, err := strconv.Atoi(tablesString)
	if err != nil || value < 1 {
		return 0, errors.New(tablesString)
	}
	return value, nil
}

// readNextTime читает следующую строку из сканнера, и пытается спарсить в time.Time 2 времени, разделенных пробелами.
// Если не получается, возвращает ошибку со считанной строкой.
func readNextTime(scanner *bufio.Scanner) (time.Time, time.Time, error) {
	end := scanner.Scan()
	if !end {
		return time.Time{}, time.Time{}, errors.New("not enough lines in file")
	}
	line := scanner.Text()

	timesStr := strings.Fields(line)

	if len(timesStr) != 2 {
		return time.Time{}, time.Time{}, errors.New(line)
	}
	startTime, err := time.Parse(time.TimeOnly, timesStr[0]+":00")
	if err != nil {
		return time.Time{}, time.Time{}, errors.New(line)
	}
	endTime, err := time.Parse(time.TimeOnly, timesStr[1]+":00")
	if err != nil {
		return time.Time{}, time.Time{}, errors.New(line)

	}
	return startTime, endTime, nil
}

func getMessage(t time.Time, code int, message string) string {
	return fmt.Sprintf("%02d:%02d %d %s", t.Hour(), t.Minute(), code, message)
}

// processEvent обрабатывает строку и возвращает ответ.
func (c *ComputerClub) processEvent(event Event) string {
	switch event.ID {
	case incomingClientCame:
		if !c.isOpen(event.T) {
			return getMessage(event.T, outcomingError, notOpenMsg)
		}
		if _, ok := c.Clients[event.ClientName]; ok {
			return getMessage(event.T, outcomingError, alreadyPresentMsg)
		}
		c.Clients[event.ClientName] = &ClientInfo{}

	case incomingClientTookTable:
		if c.isTableTaken(event.Table) {
			return getMessage(event.T, outcomingError, placeIsBusyMsg)
		}
		info, ok := c.Clients[event.ClientName]
		if !ok {
			return getMessage(event.T, outcomingError, clientUnknownMsg)
		}
		if info.AtTable {
			c.clientLeavesTable(event.ClientName, event.T)
		}
		c.clientTakesTable(event.ClientName, event.Table, event.T)

	case incomingClientWaiting:
		// если клиент уже за столом, игнорируем
		info, ok := c.Clients[event.ClientName]
		if !ok || info.AtTable {
			return ""
		}
		if c.TablesTaken < c.TablesNumber {
			return getMessage(event.T, outcomingError, canWaitNoLongerMsg)
		}
		if len(c.WaitingQueue)+1 > c.TablesNumber {
			delete(c.Clients, event.ClientName)
			return getMessage(event.T, outcomingClientLeft, event.ClientName)
		}
		c.WaitingQueue = append(c.WaitingQueue, event.ClientName)

	case incomingClientLeft:
		info, ok := c.Clients[event.ClientName]
		if !ok {
			return getMessage(event.T, outcomingError, clientUnknownMsg)
		}
		// если ушедший клиент сидел за столом
		if info.AtTable {
			c.clientLeavesTable(event.ClientName, event.T)
			if len(c.WaitingQueue) != 0 {
				nextClient := c.WaitingQueue[0]
				c.WaitingQueue = c.WaitingQueue[1:]
				c.clientTakesTable(nextClient, info.Table, event.T)

				delete(c.Clients, event.ClientName)
				return getMessage(event.T, outcomingClientTookTable, fmt.Sprintf("%s %d", nextClient, info.Table))
			}
		}
		delete(c.Clients, event.ClientName)
		contains, index := Contains(c.WaitingQueue, event.ClientName)
		if contains {
			c.WaitingQueue = append(c.WaitingQueue[:index], c.WaitingQueue[index+1:]...)
		}
	}
	return ""
}

// isOpen проверяет, находится ли вермя t в промежутке от открытия и закрытия клуба
func (c *ComputerClub) isOpen(t time.Time) bool {
	return (t.Equal(c.StartTime) || t.After(c.StartTime)) && t.Before(c.EndTime)
}

func (c *ComputerClub) isTableTaken(table int) bool {
	contains, _ := Contains(c.getListOfTakenTables(), table)
	return contains
}

func (c *ComputerClub) getListOfTakenTables() []int {
	result := []int{}
	for _, info := range c.Clients {
		if info.AtTable {
			result = append(result, info.Table)
		}
	}
	return result
}

// clientTakesTable назначет клиенту занятый стол, увеличивает счетчик занятых столов. Если клиент не находится в клубе - ничего не происходит.
func (c *ComputerClub) clientTakesTable(clientName string, tableNumber int, t time.Time) {
	info, ok := c.Clients[clientName]
	if !ok {
		return
	}

	info.AtTable = true
	info.Table = tableNumber
	info.Time = t
	c.TablesTaken++
}

// clientLeavesTable обновляет статистику по столам, уменьшает счетчик занятых столов
// если клиента нет в клубе или клиент не сидел за столом, ничего не происходит
func (c *ComputerClub) clientLeavesTable(clientName string, t time.Time) {
	info, ok := c.Clients[clientName]
	if !ok || !info.AtTable {
		return
	}
	info.AtTable = false
	c.TablesTaken--

	timeDiff := t.Sub(info.Time)
	index := info.Table - 1

	// увеличиваем время, проведенное за столом
	c.TableStats[index].TimeSpent = c.TableStats[index].TimeSpent.Add(timeDiff)

	//увеличиваем доход
	revenue := int(math.Ceil(timeDiff.Hours()))
	c.TableStats[index].Revenue += revenue * c.Tariff
}

func Contains[T comparable](slice []T, value T) (bool, int) {
	for i, v := range slice {
		if v == value {
			return true, i
		}
	}
	return false, 0
}
