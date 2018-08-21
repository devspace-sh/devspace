package fsutil

import (
	"os"
	"testing"
	//"time"
	"math/rand"
	"strings"
)

func TestWriteToFileAndReadFile(t *testing.T) {
	
	//Let's create a new file and check if the content is correct. 

	randomString := RandStringRunes(10)

	writeData := []byte("Content " + randomString)
	fileName := "C:\\Users\\covexo\\tempFolderForGoTests\\" + randomString + "\\" + randomString
	
	e := WriteToFile(writeData, fileName)

	if e != nil {
		t.Error("Write a new file failed with error: ")
		t.Error(e)
		t.Fail()
	}

	//There should be 18 bytes in the file. We'll only read 17 to test out whether this method reads the correct amount of bytes.
	readedData, e := ReadFile(fileName, 17) 

	if e != nil {
		t.Error("Reading a file failed with error: ")
		t.Error(e)
		t.Fail()
	}

	for n, byte := range readedData {

		if n >= 17 {
			t.Error("Too many bytes readed. Expected 17 bytes but actual length is: " + string(len(readedData)))
		}

		if byte != writeData[n] {
			t.Error("WriteData and ReadData don't match.\nWriteData: " + string(writeData) + "\nReadData: " + string(readedData))
			t.Fail()
			break
		}

	}

	//Now let's overwrite the content

	newData := []byte("New Content " + randomString)

	WriteToFile(newData, fileName)

	//Read everything
	readedData, e = ReadFile(fileName, -1)

	if e != nil {
		t.Error("Reading a file failed with error: ")
		t.Error(e)
		t.Fail()
	}

	for n, byte := range readedData {

		if byte != newData[n] {
			t.Error("WriteData and ReadData don't match.\nWriteData: " + string(newData) + "\nReadData: " + string(readedData))
			t.Fail()
			break
		}

	}

}

func TestCopy(t *testing.T) {

	randomString := RandStringRunes(10)
	sourcePath := "C:\\Users\\covexo\\tempFolderForGoTests\\" + randomString + "\\" + randomString

	randomString = RandStringRunes(10)
	destPath := "C:\\Users\\covexo\\tempFolderForGoTests\\" + randomString + "\\" + randomString

	WriteToFile([]byte{}, sourcePath)

	Copy(sourcePath, destPath)

}

func TestGetHomeDir(t *testing.T) {
	
	homeDirByMethod := GetHomeDir()
	homeDirByOS := os.Getenv("HOME")
	if homeDirByOS == "" {
		homeDirByOS = os.Getenv("USERPROFILE")
	}

	if homeDirByMethod != homeDirByOS {
		t.Error("Given Home Dir is wrong.\nExpected: " + homeDirByOS + " Actual: " + homeDirByMethod)
		t.Fail()
	}
}

func TestGetCurrentGofileDir(t *testing.T) {

	currentGofileDirByMethod := GetCurrentGofileDir()
	expected := os.Getenv("GOPATH") + "\\src\\git.covexo.com\\covexo\\devspace\\pkg\\util\\fsutil"

	if currentGofileDirByMethod != expected && currentGofileDirByMethod != strings.Replace(expected, "\\", "/", -1){
		t.Error("CurrentGoFileDir is not correct.\nMethod result: " + currentGofileDirByMethod + 
		"\nExpected: " + expected + 
		"\nExpected with /-separator: " + strings.Replace(expected, "\\", "/", -1))
		t.Fail()
	}
}

func TestGetCurrentGofile(t *testing.T) {

	currentGofileByMethod := GetCurrentGofile()
	expected := os.Getenv("GOPATH") + "\\src\\git.covexo.com\\covexo\\devspace\\pkg\\util\\fsutil\\filesystem_test.go"

	if currentGofileByMethod != expected && currentGofileByMethod != strings.Replace(expected, "\\", "/", -1){
		t.Error("CurrentGoFile is not correct.\nMethod result: " + currentGofileByMethod + 
		"\nExpected: " + expected + 
		"\nExpected with /-separator: " + strings.Replace(expected, "\\", "/", -1))
		t.Fail()
	}
}

//Method for random letter string
var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
    b := make([]rune, n)
    for i := range b {
        b[i] = letterRunes[rand.Intn(len(letterRunes))]
    }
    return string(b)
}
