package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

type TestCase struct {
	inputFile  string
	outputFile string
}

const (
	resourceFolder = "testdata"
)

func getFullPath(filename string) string {
	return filepath.Join(resourceFolder, filename)
}

// compareLines разбивает строки по символам переноса строки и сравнивает каждую строку
func compareLines(str1, str2 string) bool {
	//позволяет избежать ошибок сравнения из-за символа возврата коретки
	lines1 := regexp.MustCompile(`(\n)|(\r\n)`).Split(str1, -1)
	lines2 := regexp.MustCompile(`(\n)|(\r\n)`).Split(str2, -1)

	if len(lines1) != len(lines2) {
		return false
	}
	for i := range lines1 {
		if (lines1[i]) != lines2[i] {
			fmt.Printf("not equal: %q %q\n", lines1[i], lines2[i])
			return false
		}
	}
	return true
}

func TestRun(t *testing.T) {
	testCases := []TestCase{
		// базовый пример
		{
			inputFile:  getFullPath("1_in.txt"),
			outputFile: getFullPath("1_out.txt"),
		},
		// проверяются различные сообщения об ошибках
		{
			inputFile:  getFullPath("2_in.txt"),
			outputFile: getFullPath("2_out.txt"),
		},
		// некорректный формат:

		{
			inputFile:  getFullPath("3_in.txt"),
			outputFile: getFullPath("3_out.txt"),
		},
		{
			inputFile:  getFullPath("4_in.txt"),
			outputFile: getFullPath("4_out.txt"),
		},
		// нет имени клиента
		{
			inputFile:  getFullPath("5_in.txt"),
			outputFile: getFullPath("5_out.txt"),
		},
		// следующее событие - раньше по времени
		{
			inputFile:  getFullPath("6_in.txt"),
			outputFile: getFullPath("6_out.txt"),
		},
		// неверный id события
		{
			inputFile:  getFullPath("7_in.txt"),
			outputFile: getFullPath("7_out.txt"),
		},
		// некорректное имя клиента
		{
			inputFile:  getFullPath("8_in.txt"),
			outputFile: getFullPath("8_out.txt"),
		},
		// некорректный номер стола
		{
			inputFile:  getFullPath("9_in.txt"),
			outputFile: getFullPath("9_out.txt"),
		},
		// пустой файл
		{
			inputFile:  getFullPath("10_in.txt"),
			outputFile: getFullPath("10_out.txt"),
		},
	}

	for _, testCase := range testCases {
		var b bytes.Buffer
		stream := bufio.NewReadWriter(
			bufio.NewReader(&b),
			bufio.NewWriter(&b),
		)
		Run(testCase.inputFile, stream)
		stream.Flush()
		output, err := io.ReadAll(stream)
		if err != nil {
			panic(err)
		}
		bytes, err := os.ReadFile(testCase.outputFile)
		if err != nil {
			panic(err)
		}
		outputFileContents := string(bytes)
		if !compareLines(string(output), outputFileContents) {
			t.Errorf("for input from file %s expected output:\n%s\ngot:\n%s", testCase.inputFile, outputFileContents, output)
		}
	}
}
