package log

/*func TestGetLogger(t *testing.T) {

	fsutil.WriteToFile(make([]byte, 0), "./.devspace/logs/TestLogger.log")

	logger := GetLogger("TestLogger", true)

	logger.Info("Some Test Log")
	logger.Warn("More Logs")

	fileContent, err := fsutil.ReadFile("./.devspace/logs/TestLogger.log", -1)

	if err != nil {
		t.Error("Error while reading Logfile")
		t.Fail()
	}

	t.Logf(string(fileContent))

	logsAsStrings := strings.Split(string(fileContent), "}")
	logsAsStructs := make([]Log, len(logsAsStrings))

	for n, logAsString := range logsAsStrings {

		if n == len(logsAsStrings)-1 {
			break
		}

		json.Unmarshal([]byte(logAsString+"}"), &logsAsStructs[n])
	}

	if logsAsStructs[0].Level != "info" || logsAsStructs[1].Level != "warning" {
		t.Error("Logs aren't shown as info and warning. Instead they are: " + logsAsStructs[0].Level + " and " + logsAsStructs[1].Level)
		t.Fail()
	}

	if logsAsStructs[0].Msg != "Some Test Log" || logsAsStructs[1].Msg != "More Logs" {
		t.Error("Wrong messages in logs.\nMessage 1: " + logsAsStructs[0].Msg +
			"\nExpected: Some Test Log" +
			"\nMessage 2: " + logsAsStructs[1].Msg +
			"\nExpected: More Logs")
		t.Fail()
	}
} */
