package main

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ttacon/chalk"
	"github.com/xuri/excelize/v2"
)

var countTimeout int = 0

type StatusResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func toBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func saveCsv(file *os.File, data []string) {
	w := csv.NewWriter(file)
	defer w.Flush()
	err := w.Write(data)
	checkError("No se puede escribir", err)
}

func checkError(message string, err error) {
	if err != nil {
		log.Fatal(chalk.Red, message, chalk.Red, err)
	}
}

func showLog(message string) {
	log.Println(chalk.Green, ">> ", chalk.White, message)
}

func readExcel(file *os.File) {
	sheetName := "Hoja1"
	f, err := excelize.OpenFile("./lista.xlsx")
	checkError("No se puede leer el archivo lista.xlsx", err)

	rows, err := f.GetRows(sheetName)
	checkError("No hay datos lista.xlsx", err)

	for index := range rows {
		fmt.Println(chalk.Blue, "[LEN] ", chalk.White, strconv.Itoa(index+1)+"/"+strconv.Itoa(len(rows)))
		cellPhone, err := f.GetCellValue(sheetName, ("A" + strconv.Itoa(index+1)))
		checkError("No se pudo leer la celda A"+strconv.Itoa(index+1), err)
		cellName, err2 := f.GetCellValue(sheetName, ("B" + strconv.Itoa(index+1)))
		checkError("No se pudo leer la celda B"+strconv.Itoa(index+1), err2)
		cellMessage, err3 := f.GetCellValue(sheetName, ("C" + strconv.Itoa(index+1)))
		checkError("No se pudo leer la celda C"+strconv.Itoa(index+1), err3)
		isImage, err4 := f.GetCellValue(sheetName, ("D" + strconv.Itoa(index+1)))
		checkError("No se pudo leer la celda D"+strconv.Itoa(index+1), err4)

		if isImage == "1" {
			fmt.Println(chalk.Green, "[INF] ", chalk.White, cellPhone, cellName, chalk.Yellow, "Message Image")
			sendMessageWhatsApp(file, cellPhone, cellName, cellMessage, "http://localhost:5004/message/send/image")
		} else {
			fmt.Println(chalk.Green, "[INF] ", chalk.White, cellPhone, cellName, chalk.Yellow, "Message Text")
			sendMessageWhatsApp(file, cellPhone, cellName, cellMessage, "http://localhost:5004/message/send")
		}
		time.Sleep(10 * time.Second)
	}
}

func sendMessageWhatsApp(file *os.File, phone string, image string, message string, url string) {
	phone = strings.TrimSpace("591" + phone)
	imageConvert := image
	if image != "" {
		bytes, err := ioutil.ReadFile("./file.jpeg")
		if err != nil {
			log.Fatal(err)
		}
		imageConvert = toBase64(bytes)
	}
	message = strings.Replace(message, "\\n", "\n", -1)
	values := map[string]string{"phone": phone, "image": imageConvert, "caption": message, "message": message}
	json_data, err := json.Marshal(values)
	checkError("Error al parsear en JSON", err)

	var statusStruct StatusResponse
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Post(url, "application/json", bytes.NewBuffer(json_data))

	if err != nil {
		statusStruct.Status = "FAILED"
		countTimeout++
	} else {
		countTimeout = 0
		bytes, errReader := ioutil.ReadAll(response.Body)
		defer response.Body.Close()
		checkError("No se pudo leer la respuesta", errReader)
		errJson := json.Unmarshal(bytes, &statusStruct)
		checkError("Error al formatear JSON", errJson)
	}

	if statusStruct.Status == "SUCCESS" {
		log.Println(chalk.Green, "[OK] "+phone+" "+statusStruct.Message)
	} else if statusStruct.Status == "FAILED" {
		log.Println(chalk.Red, "[FAIL] "+phone+" no se pudo enviar TIMEOUT")
		saveCsv(file, []string{phone, "NO SE ENVIO"})
		if countTimeout >= 3 {
			log.Fatal(chalk.Red, "STOP SCRIPT - SERVER IS DOWN")
		}
	} else {
		log.Println(chalk.Red, "[ERR] "+phone+" "+statusStruct.Status)
		saveCsv(file, []string{phone, statusStruct.Status})
		// log.Fatal(chalk.Red, phone, chalk.Red, image)
	}
}

func main() {
	showLog("Iniciando script")
	t := time.Now().UnixNano() / 1000000
	nameFileCsv := strconv.FormatInt(int64(t), 10) + "_phoneInvalid.csv"
	filePhoneInvalid, err := os.Create(nameFileCsv)
	checkError("No se puede crear el archivo", err)

	readExcel(filePhoneInvalid)
}
