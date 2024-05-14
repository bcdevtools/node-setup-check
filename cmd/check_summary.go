package cmd

type checkRecord struct {
	fatal   bool
	message string
	suggest string
	addedNo int
}

var checkRecords []checkRecord

func putCheckRecord(record checkRecord) {
	record.addedNo = len(checkRecords) + 1
	checkRecords = append(checkRecords, record)
}

func fatalRecord(message string, suggest string) {
	putCheckRecord(checkRecord{fatal: true, message: message, suggest: suggest})
}

func warnRecord(message string, suggest string) {
	putCheckRecord(checkRecord{fatal: false, message: message, suggest: suggest})
}
