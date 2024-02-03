package service

import (
	"archive/zip"
	"bytes"
	"erinyes/builder"
	"erinyes/logs"
	"erinyes/parser"
	"github.com/gin-gonic/gin"
	"net/http"
	"os/exec"
	"time"
)

func HandlePing(c *gin.Context) {
	c.String(http.StatusOK, "Welcome to use graph build service")
}

type GeneralLogData struct {
	Log string `json:"log"`
}

type GeneralLogsData struct {
	Logs []string `json:"logs"`
}

func HandleSysdigLog(c *gin.Context) {
	var sysdigData GeneralLogData
	if err := c.ShouldBindJSON(&sysdigData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	parser.SysdigRawChan <- sysdigData.Log
	c.String(http.StatusOK, "Add sysdig log to chan success")
}

func HandleSysdigLogs(c *gin.Context) {
	var sysdigData GeneralLogsData
	if err := c.ShouldBindJSON(&sysdigData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for _, value := range sysdigData.Logs {
		parser.SysdigRawChan <- value
	}

	c.String(http.StatusOK, "Add all sysdig logs to chan success")
}

func HandleNetLog(c *gin.Context) {
	var netData GeneralLogData
	if err := c.ShouldBindJSON(&netData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	parser.NetRawChan <- netData.Log
	c.String(http.StatusOK, "Add net log to chan success")
}

func HandleNetLogs(c *gin.Context) {
	var netData GeneralLogsData
	if err := c.ShouldBindJSON(&netData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for _, value := range netData.Logs {
		parser.NetRawChan <- value
	}

	c.String(http.StatusOK, "Add all net logs to chan success")
}

func HandleGenerate(c *gin.Context) {
	currentTime := time.Now()
	currentTimeString := currentTime.Format("20060102150405")
	dotName := currentTimeString + ".dot"
	svgName := currentTimeString + ".svg"
	dotString := builder.GenerateDotGraph("").String()
	dotContent := []byte(dotString)
	err, svgContent := generateSVGFromDot(dotContent)
	if err != nil {
		logs.Logger.WithError(err).Errorf("generate svg failed")
		c.String(http.StatusInternalServerError, "generate svg failed")
		return
	}

	var zipBuffer bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuffer)

	dotFile, _ := zipWriter.Create(dotName)
	dotFile.Write(dotContent)

	svgFile, _ := zipWriter.Create(svgName)
	svgFile.Write(svgContent)

	zipWriter.Close()

	c.Header("Content-Disposition", "attachment; filename=files.zip")

	c.Data(http.StatusOK, "application/zip", zipBuffer.Bytes())
}

func generateSVGFromDot(dotContent []byte) (error, []byte) {
	cmd := exec.Command("dot", "-Tsvg")
	cmd.Stdin = bytes.NewReader(dotContent)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return err, []byte{}
	}
	return nil, out.Bytes()
}
