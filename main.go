package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	iClientCame      = 1
	iClientTookTable = 2
	iClientWaiting   = 3
	iClientLeft      = 4

	oClientLeft      = 11
	oClientTookTable = 12
	oError           = 13

	notOpenMsg         = "NotOpenYet"
	alreadyPresentMsg  = "YouShallNotPass"
	placeIsBusyMsg     = "PlaceIsBusy"
	clientUnknownMsg   = "ClientUnknown"
	canWaitNoLongerMsg = "ICanWaitNoLonger"
)

type ComputerClub struct {
	TablesNumber int
	StartTime    time.Time
	EndTime      time.Time
	Price        int

	TablesTaken int

	Clients      map[string]*ClientInfo
	WaitingQueue []string // how to remove clients from here?
}

type ClientInfo struct {
	AtTable bool
	Table   int
}

// видимо надо не парсить строку за строкой, а сначала рапарсить весь файл, получить список ивентов, а потом уже выводить
// если во время парсинга возникла ошибка - выводить строку.
func main() {
	// args := os.Args
	// if len(args) < 2 {
	// 	fmt.Println("please provide a file path as first argument")
	// 	return
	// }
	// filePath := args[1]
	filePath := "input2.txt"

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("error opening file:", err)
		return
	}

	club := ComputerClub{
		Clients: make(map[string]*ClientInfo),
	}

	scanner := bufio.NewScanner(file)

	club.TablesNumber, err = readNextInt(scanner)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	club.StartTime, club.EndTime, err = readNextTime(scanner)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	club.Price, err = readNextInt(scanner)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Printf("%02d:%02d\n", club.StartTime.Hour(), club.StartTime.Minute())
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
		err := club.processInput(line)
		if err != nil {
			return
		}
	}
	fmt.Printf("%02d:%02d\n", club.EndTime.Hour(), club.EndTime.Minute())
}

// readNextInt читает следующую строку из сканнера, и пытается конвертировать в int. Если не получается, возвращает ошибку со считанной строкой.
func readNextInt(scanner *bufio.Scanner) (int, error) {
	end := scanner.Scan()
	if !end {
		return 0, errors.New("not enough lines in file")
	}
	tablesString := scanner.Text()
	value, err := strconv.Atoi(tablesString)
	if err != nil {
		return 0, errors.New(tablesString)
	}
	return value, nil
}

// readNextTime читает следующую строку из сканнера, и пытается спарсить в time.Time 2 времени, разделенных пробелами. Если не получается, возвращает ошибку со считанной строкой.
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
		fmt.Println("error:", err)
		return time.Time{}, time.Time{}, errors.New(line)
	}
	endTime, err := time.Parse(time.TimeOnly, timesStr[1]+":00")
	if err != nil {
		fmt.Println("error:", err)
		return time.Time{}, time.Time{}, errors.New(line)

	}
	return startTime, endTime, nil
}

func printMessage(t time.Time, code int, message string) {
	fmt.Printf("%02d:%02d %d %s\n", t.Hour(), t.Minute(), code, message)
}

// processInput обрабатывает строку и выводит сообщение. Если возникает ошибка, возвращает error.
func (c *ComputerClub) processInput(input string) error {
	params := strings.Fields(input)
	if len(params) < 3 {
		return errors.New("invalid line")
	}

	time, err := time.Parse(time.TimeOnly, params[0]+":00")
	if err != nil {
		return err
	}
	event, err := strconv.Atoi(params[1])
	if err != nil {
		return err
	}
	clientName := params[2]

	switch event {
	case iClientCame:
		if !c.isOpen(time) {
			printMessage(time, oError, notOpenMsg)
			return nil
		}
		if _, ok := c.Clients[clientName]; ok {
			printMessage(time, oError, alreadyPresentMsg)
			return nil
		}
		c.Clients[clientName] = &ClientInfo{}

	case iClientTookTable:
		if len(params) != 4 {
			return errors.New("invalid line")
		}
		tableNumber, err := strconv.Atoi(params[3])
		if err != nil {
			return err
		}
		if tableNumber > c.TablesNumber || tableNumber < 1 {
			return errors.New("incorrect table number")
		}
		if c.isTableTaken(tableNumber) {
			printMessage(time, oError, placeIsBusyMsg)
			return nil
		}
		if _, ok := c.Clients[clientName]; !ok {
			printMessage(time, oError, clientUnknownMsg)
			return nil
		}
		c.takeTable(clientName, tableNumber)

	case iClientWaiting:
		if c.TablesTaken < c.TablesNumber {
			printMessage(time, oError, canWaitNoLongerMsg)
			return nil
		}
		if len(c.WaitingQueue)+1 > c.TablesNumber {
			printMessage(time, oClientLeft, clientName)
			return nil
		}
		c.WaitingQueue = append(c.WaitingQueue, clientName)

	case iClientLeft:
		info, ok := c.Clients[clientName]
		if !ok {
			printMessage(time, oError, clientUnknownMsg)
			return nil
		}
		delete(c.Clients, clientName)
		// если ушедший клиент сидел за столом
		if info.AtTable {
			if len(c.WaitingQueue) != 0 {
				nextClient := c.WaitingQueue[0]
				c.WaitingQueue = c.WaitingQueue[1:]
				c.Clients[nextClient] = &ClientInfo{
					AtTable: true,
					Table:   info.Table,
				}
				printMessage(time, oClientTookTable, nextClient)
			} else {
				c.TablesTaken--
			}
		}
	}

	return nil
}

// isOpen проверяет, находится ли вермя t в промежутке от открытия и закрытия клуба
func (c *ComputerClub) isOpen(t time.Time) bool {
	return t.After(c.StartTime) && t.Before(c.EndTime)
}

func (c *ComputerClub) isTableTaken(table int) bool {
	for _, info := range c.Clients {
		if info.AtTable && info.Table == table {
			return true
		}
	}
	return false
}

// takeTable назначет клиенту занятый стол, если клиент до этого не сидел за столом увеличивает счетчик занятых столов. Если клиент не находится в клубе - ничего не происходит.
func (c *ComputerClub) takeTable(clientName string, tableNumber int) {
	info, ok := c.Clients[clientName]
	if !ok {
		return
	}

	info.Table = tableNumber
	if !info.AtTable {
		info.AtTable = true
		c.TablesTaken++
	}
}
