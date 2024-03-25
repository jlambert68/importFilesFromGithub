package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Struct for parsing JSON response
type GitHubFile struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // "file" or "dir"
	URL     string `json:"url"`  // URL to fetch contents if it's a directory
	Content []byte `json:"content"`
}
type GitHubFileDetail struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	URL         string `json:"url"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
	// Include other fields as needed
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("File Picker Example & GitHub File Picker")

	button := widget.NewButton("Open File", func() {
		filePicker := dialog.NewFileOpen(
			func(r fyne.URIReadCloser, err error) {
				if err == nil && r != nil {
					// Handle file read here
					defer r.Close()
				}
			}, myWindow)

		filePicker.SetFilter(storage.NewExtensionFileFilter([]string{".png", ".jpg", ".txt"})) // Set file filter if needed
		filePicker.Show()
	})

	repoOwner := "jlambert68"    // Replace with the repository owner's username
	repoName := "FenixTesterGui" // Replace with the repository name
	repoPath := ""               // Replace with the path in the repository, if any

	originalApiUrl := "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/contents" + repoPath
	apiUrl := originalApiUrl

	var githubFiles, selectedFiles []GitHubFile
	githubFiles = getFileListFromGitHub(apiUrl)

	// Create a string data binding
	var currentPath binding.String

	selectedFilesList := widget.NewList(
		func() int { return len(selectedFiles) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(selectedFiles[i].Name)
		},
	)

	selectedFilesList.OnSelected = func(id widget.ListItemID) {
		// Remove the file from selectedFiles and refresh the list
		selectedFiles = append(selectedFiles[:id], selectedFiles[id+1:]...)
		selectedFilesList.Unselect(id)
		selectedFilesList.Refresh()
	}

	var fileList *widget.List
	fileList = widget.NewList(
		func() int {
			return len(githubFiles)
		},
		func() fyne.CanvasObject {
			// Create a CustomLabel for each item.
			label := NewCustomLabel("Template", func() {
				// Define double-click action here.
			})
			return label
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			// Update the label text and double-click action for each item.
			label := obj.(*CustomLabel)
			label.Text = githubFiles[id].Name

			label.onDoubleTap = func() {

				selectedFile := githubFiles[id]
				if selectedFile.Type == "dir" {
					// The item is a directory; fetch its contents
					githubFiles = getFileListFromGitHub(selectedFile.URL)
					fileList.Refresh() // Refresh the list to update it with the new contents
					currentPath.Set(strings.Split(selectedFile.URL, "?")[0])

					apiUrl = selectedFile.URL
				} else if selectedFile.Type == "file" {
					// Add file to selectedFiles and refresh the list only when if it doesn't exist
					var shouldAddFile bool
					shouldAddFile = true
					for _, existingSelectedFile := range selectedFiles {
						if existingSelectedFile.URL == selectedFile.URL {
							shouldAddFile = false
							break
						}
					}

					if shouldAddFile == true {
						selectedFiles = append(selectedFiles, selectedFile)
						selectedFilesList.Refresh()

					}

				} else {
					// Show a dialog when other.
					dialog.ShowInformation("Info", "Double-clicked on: "+githubFiles[id].Name+" with Type "+githubFiles[id].Type, myWindow)
				}
			}
			label.Refresh()
		},
	)

	//currentPath = binding.NewString()
	currentPath = binding.NewString()
	currentPath.Set(strings.Split(apiUrl, "?")[0]) // Setting initial value

	// Create a label with data binding
	var pathLabel *widget.Label
	pathLabel = widget.NewLabelWithData(currentPath)
	/*
		pathLabel2 := NewTappableLabel(&currentPath, func() {
			// Code to move up one directory
			// Example: if currentPath is "/a/b/c/", new path should be "/a/b/"
			currentPathAsString, err := currentPath.Get()
			if err != nil {
				log.Fatalln(err)
			}
			newPath, err := MoveUpInPath(currentPathAsString)
			if err != nil {
				// Handle error (maybe already at root)
			} else {
				//currentPath = newPath
				//pathLabel.SetText("Current Path: " + currentPath)
				currentPath.Set(newPath)
				// Refresh file list for new path
				githubFiles = getFileListFromGitHub(newPath)
				fileList.Refresh()
			}
		})


	*/
	//backArrowIcon := widget.NewIcon(theme.NavigateBackIcon())

	// You can make the icon clickable by placing it in a button
	backButton := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		// Handle the button click - go back in your navigation, for instance

		if apiUrl == originalApiUrl {
			return
		}

		newPath, err := MoveUpInPath(apiUrl)
		if err == nil || len(newPath) > 0 {
			apiUrl = newPath

			currentPath.Set(strings.Split(apiUrl, "?")[0])
			githubFiles = getFileListFromGitHub(apiUrl)
			fileList.Refresh() // Refresh the list to update it with the new contents

		}
	})

	importButton := widget.NewButton("Import Files", func() {
		for fileIndex, file := range selectedFiles {
			content, err := loadFileContent(file)
			if err != nil {
				dialog.ShowError(err, myWindow)
				continue
			}
			// Do something with the content, e.g., display it, process it, etc.
			selectedFiles[fileIndex].Content = content

			extractedContent, err := extractContentFromJson(string(content))
			if err != nil {
				log.Fatalf("Error parsing JSON: %s", err)
			}
			fmt.Println("Extracted content:", extractedContent)

			contentAsString, err := decodeBase64Content(string(extractedContent))
			if err != nil {
				log.Fatalf("Failed to decode content: %s", err)
			}
			// 'content' now contains the decoded file content as a string
			fmt.Println(contentAsString)
			//playBeep()

		}
	})

	cancelButton := widget.NewButton("Cancel", func() {
		myWindow.Close()
	})

	myWindow.Resize(fyne.NewSize(400, 500)) // Set initial size of the window

	var pathRow *fyne.Container
	pathRow = container.NewHBox(backButton, pathLabel)

	myTopLayout := container.NewVBox(button, pathRow) //, pathLabel2)

	splitContainer := container.NewHSplit(fileList, selectedFilesList)
	splitContainer.Offset = 0.5 // Adjust if you want different initial proportions

	var importCancelRow *fyne.Container
	importCancelRow = container.NewHBox(layout.NewSpacer(), importButton, cancelButton)

	content := container.NewBorder(myTopLayout, importCancelRow, nil, nil, splitContainer)
	myWindow.SetContent(content)
	myWindow.ShowAndRun()
}

func playBeep() {
	switch runtime.GOOS {
	case "windows":
		exec.Command("rundll32", "user32.dll,MessageBeep").Run()
	case "darwin":
		exec.Command("osascript", "-e", "beep").Run()
	case "linux":
		// Linux does not have a standard beep command for GUI, this is a simple alternative
		//exec.Command("echo", "-ne", "\\007").Run()
		exec.Command("beep").Run()
	// Add more cases here if needed for other operating systems
	default:
		// Optionally handle unsupported operating systems
	}
}

func extractContentFromJson(jsonData string) (string, error) {
	var fileDetail GitHubFileDetail
	err := json.Unmarshal([]byte(jsonData), &fileDetail)
	if err != nil {
		return "", err
	}

	return fileDetail.Content, nil
}

func decodeBase64Content(encodedContent string) (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedContent)
	if err != nil {
		return "", err
	}
	return string(decodedBytes), nil
}

func loadFileContent(file GitHubFile) ([]byte, error) {
	// Assuming file.URL is the URL to the raw content of the file
	resp, err := http.Get(file.URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch file: %s", resp.Status)
	}

	return ioutil.ReadAll(resp.Body)
}

func MoveUpInPath(currentPath string) (string, error) {
	// Trim any trailing slashes
	trimmedPath := strings.TrimRight(currentPath, "/")

	// Split the path into components
	pathComponents := strings.Split(trimmedPath, "/")

	// Check if it's already at the root or has no parent
	if len(pathComponents) <= 1 {
		return "", fmt.Errorf("already at the root or invalid path")
	}

	// Remove the last component to move up one directory
	newPathComponents := pathComponents[:len(pathComponents)-1]

	// Join components back into a path
	newPath := strings.Join(newPathComponents, "/")
	if newPath == "" {
		newPath = "/" // Ensure root is represented correctly
	}

	return newPath, nil
}

func getFileListFromGitHub(apiUrl string) []GitHubFile {

	client := &http.Client{}
	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		log.Fatalf("Error creating request: %s", err.Error())
	}

	// Add the API token in the request header
	apiToken := gitHubApiKey // Replace with your actual API token
	req.Header.Add("Authorization", "token "+apiToken)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error occurred while calling GitHub API: %s", err.Error())
	}
	defer resp.Body.Close()
	if err != nil {
		log.Fatalf("Error reading API response: %s", err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading API response: %s", err.Error())
	}

	var files []GitHubFile
	if err := json.Unmarshal(body, &files); err != nil {
		log.Fatalf("Error unmarshalling JSON: %s", err.Error())
	}

	return files
}

type CustomLabel struct {
	widget.Label
	onDoubleTap func()
	lastTap     time.Time
}

func NewCustomLabel(text string, onDoubleTap func()) *CustomLabel {
	l := &CustomLabel{Label: widget.Label{Text: text}, onDoubleTap: onDoubleTap}
	l.ExtendBaseWidget(l)
	return l
}

func (l *CustomLabel) Tapped(e *fyne.PointEvent) {
	now := time.Now()
	if now.Sub(l.lastTap) < 500*time.Millisecond { // 500 ms as double-click interval
		if l.onDoubleTap != nil {
			l.onDoubleTap()
		}
	}
	l.lastTap = now
}

func (l *CustomLabel) TappedSecondary(*fyne.PointEvent) {
	// Implement if you need right-click (secondary tap) actions.
}

func (l *CustomLabel) MouseIn(*desktop.MouseEvent)    {}
func (l *CustomLabel) MouseMoved(*desktop.MouseEvent) {}
func (l *CustomLabel) MouseOut()                      {}

type TappableLabel struct {
	widget.Label
	onDoubleTap func()
	lastTap     time.Time
}

func NewTappableLabel(text *binding.String, onDoubleTap func()) *TappableLabel {
	labelWithData := widget.NewLabelWithData(*text)

	l := &TappableLabel{
		Label:       *labelWithData,
		onDoubleTap: onDoubleTap,
	}

	//l.Label = *widget.NewLabelWithData(text)

	l.ExtendBaseWidget(l)
	return l
}

func (l *TappableLabel) Tapped(e *fyne.PointEvent) {
	now := time.Now()
	if now.Sub(l.lastTap) < 500*time.Millisecond { // 500 ms as double-click interval
		if l.onDoubleTap != nil {
			l.onDoubleTap()
		}
	}
	l.lastTap = now
}
