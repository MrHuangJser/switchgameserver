package helper

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type FileIndex struct {
	Size int    `json:"size"`
	Url  string `json:"url"`
}

type FilesIndex struct {
	Files   []FileIndex `json:"files"`
	Success string      `json:"success"`
}

var driveService *drive.Service = nil

func initDriveService() (*drive.Service, error) {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
		return nil, err
	}

	config, err := google.ConfigFromJSON(b, drive.DriveScope, drive.DriveFileScope, drive.DriveReadonlyScope, drive.DriveMetadataReadonlyScope, drive.DriveMetadataScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	token := GetToken(config)

	srv, err := drive.NewService(context.Background(), option.WithTokenSource(config.TokenSource(context.Background(), token)))
	if err != nil {
		log.Fatalf("Unable to init drive client: %v", err)
		return nil, err
	}
	return srv, nil
}

func getLastDrive() (*drive.Drive, error) {
	if driveService == nil {
		service, err := initDriveService()
		if err != nil {
			fmt.Printf("Unable init drive service: %v", err)
			return nil, err
		}
		driveService = service
	}

	listSrv := drive.NewDrivesService(driveService)

	drives, err := listSrv.List().PageSize(20).Do()

	if err != nil {
		fmt.Printf("Unable to get drive list: %v", err)
	}
	var lastDrive *drive.Drive
	index := 0
	for _, d := range drives.Drives {
		currentIndex, err := strconv.Atoi(regexp.MustCompile(`^hbg(\d+)`).FindStringSubmatch(d.Name)[1])
		if err != nil {
			return nil, err
		}
		if currentIndex > index {
			index = currentIndex
			lastDrive = d
		}
	}
	return lastDrive, err
}

func getAllFiles(list *[]FileIndex, driveId string, pageToken ...string) error {
	fileService := drive.NewFilesService(driveService)
	fileList, err := fileService.List().DriveId(driveId).Corpora("drive").SupportsAllDrives(true).IncludeItemsFromAllDrives(true).Spaces("drive").Fields("nextPageToken, files(id, name, size)").PageSize(1000).Q(`trashed = false and name contains '.nsz' or name contains '.nsp' or name contains '.xci'`).PageToken(pageToken[0]).Do()
	if err != nil {
		fmt.Printf("Failed to load file list: %v", err)
		return err
	}
	for _, driveFile := range fileList.Files {
		titleMatchResult := regexp.MustCompile(`(\[[0-9A-Fa-f]{16}\])`).FindStringSubmatch(driveFile.Name)
		versionMatchResult := regexp.MustCompile(`(\[v[0-9]+\])`).FindStringSubmatch(driveFile.Name)
		if len(titleMatchResult) > 1 && len(versionMatchResult) > 1 {
			titleId := titleMatchResult[1]
			version := versionMatchResult[1]
			extName := regexp.MustCompile(`\.(xci|nsz|nsp|XCI|NSZ|NSP)`).FindStringSubmatch(driveFile.Name)[1]
			url := fmt.Sprintf("gdrive:%v#%v%v.%v", driveFile.Id, titleId, version, extName)
			fileIndex := &FileIndex{Size: int(driveFile.Size), Url: url}
			*list = append(*list, *fileIndex)
		} else {
			fileIndex := &FileIndex{Size: int(driveFile.Size), Url: fmt.Sprintf("gdrive:%v", driveFile.Name)}
			*list = append(*list, *fileIndex)
		}
	}
	if len(fileList.NextPageToken) > 0 {
		getAllFiles(list, driveId, fileList.NextPageToken)
	}
	return nil
}

func genFilesIndex() error {
	lastDrive, err := getLastDrive()
	if err != nil {
		fmt.Printf("Unable to get last drive: %v", err)
		return err
	}

	var allFiles []FileIndex = make([]FileIndex, 0)

	err = getAllFiles(&allFiles, lastDrive.Id, "")

	if err != nil {
		return err
	}

	filesIndex := &FilesIndex{
		Files:   allFiles,
		Success: "enjoy hbg shop!",
	}

	f, err := os.OpenFile("./hbg.json", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Printf("Unable to generator hbg.json: %v", err)
		return err
	}
	defer f.Close()
	json.NewEncoder(f).Encode(filesIndex)
	return nil
}

func GetFilesIndex() error {
	fileStat, err := os.Stat("./hbg.json")
	if err == nil || os.IsExist(err) {
		modTime := fileStat.ModTime().Unix()
		timestamp := time.Now().Unix()
		if timestamp-modTime >= 3600*24 {
			err = genFilesIndex()
		}
	} else {
		err = genFilesIndex()
	}

	if err != nil {
		return err
	}

	return nil
}
