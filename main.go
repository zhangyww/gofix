package main

import (
	"flag"
	"os"
	"fmt"
	"path"
	"path/filepath"
	"io/ioutil"
	"strings"
	"bytes"
	"bufio"
	"io"
)


type FileState struct {
	IsImportRegion 		bool
	IsBlockComment 		bool
	IsRawStringRegion 	bool
}


var rootDir string
var moduleName string

var fileBuffer bytes.Buffer
var importPathBuffer bytes.Buffer
const MAX_PATH_LEVEL int = 32
var pathRel = [MAX_PATH_LEVEL]string{}

func main() {

	flag.StringVar(&rootDir, "dir", "./", "specify the root dir of your module")
	flag.Parse()
	
	others := flag.Args()
	if len(others) > 0 {
		fmt.Println("gofix usage:")
		flag.PrintDefaults()
		os.Exit(1)
	}
	
	if exists, _ := PathExists(rootDir); !exists {
		fmt.Fprintf(os.Stderr, "dir not exists: %s\n", rootDir)
		os.Exit(1)
	}
	
	gomodfile := path.Join(rootDir, "go.mod")
	if exists, _ := PathExists(gomodfile); !exists {
		fmt.Fprintf(os.Stderr, "go.mod file not exists.\n")
		os.Exit(1)
	}
	
	ReadModuleName(gomodfile)
	
	fmt.Printf("module : %d\n", len([]rune(moduleName)))
	//fmt.Println(flag.NArg())
	
	fileBuffer.Grow(4096)

	err := FixDir(rootDir, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error : %s\n", err)
		os.Exit(1)
	}
	
}


func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}


func ReadModuleName(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("err open go.mod file: %s", path)
	}
	defer file.Close()
	
	fileReader := bufio.NewReader(file)
	hasModuleName := false
	for {
		line, err := fileReader.ReadString('\n')
		
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		
		tokens := strings.Split(line, " ")
		
		if len(tokens) == 2 && tokens[0] == "module" {
			moduleName = tokens[1]
			moduleName = strings.Trim(moduleName, " \t\r\n")
			hasModuleName = true
		}
	}
	
	if !hasModuleName {
		return fmt.Errorf("go.mod does not specify the module field")
	}
	
	return nil
}


func FixDir(path string, pathLevel int) error {
	if pathLevel >= MAX_PATH_LEVEL {
		return fmt.Errorf("path depth level is %d (dir: %s)", MAX_PATH_LEVEL, path)
	}

	fileInfos, err := ioutil.ReadDir(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error in dir : %s\n", path)
		return err
	}

	for i := range fileInfos {
		if fileInfos[i].IsDir() {
			pathRel[pathLevel] = fileInfos[i].Name()
			FixDir(filepath.Join(path, fileInfos[i].Name()), pathLevel + 1)
		} else {
			if strings.HasSuffix(fileInfos[i].Name(), ".go") {
				FixFile(filepath.Join(path, fileInfos[i].Name()), pathLevel)
			}
		}
	}

	return nil
}



func FixFile(path string, pathLevel int) error {
	
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	fileReader := bufio.NewReader(file)
	fileBuffer.Reset()
	fileState := FileState{
		IsImportRegion: false,
		IsBlockComment: false,
		IsRawStringRegion: false,
	}
	for {
		line, err := fileReader.ReadString('\n')
		if err == io.EOF {
			//fmt.Print(line)
			break
		}
		if err != nil {
			return fmt.Errorf("unknown error")
		}

		FixLine(line, pathLevel, &fileState)
	}
	
	//改写文件
	file, err = os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	file.Write(fileBuffer.Bytes())
	fileBuffer.Reset()

	return nil
}


func FixLine(line string, pathLevel int, fileState *FileState) {
	isImportChar := false
	isImportString := false
	prevChar := rune(' ')
	isQuoteRegion := false
	backLevel := 0
	dotCount := 0
	isLineComment := false
	isStringRegion :=  false
	
	for _, c := range line {

		if fileState.IsBlockComment {
			fileBuffer.WriteRune(c)

			if prevChar == '*' && c == '/' {
				//对块注释结束 "*/"的判断
				fileState.IsBlockComment = false
			}
			prevChar = c
			continue
		} 

		if fileState.IsRawStringRegion {
			fileBuffer.WriteRune(c)

			if c == '`' {
				//raw string结束
				fileState.IsRawStringRegion = false
			}
			prevChar = c
			continue
		}

		if isLineComment {
			fileBuffer.WriteRune(c)
			//注释行
			//将每个字符都输出到缓存
			continue
		}

		if isQuoteRegion {
			// import 的双引号内容
			if c == '"' {
				isQuoteRegion = false
				importPath := importPathBuffer.String()

				FixImportPath(importPath, pathLevel)
				fileBuffer.WriteRune(c)

				importPathBuffer.Reset()
			} else {
				importPathBuffer.WriteRune(c)
			}
			prevChar = c
			continue
		}

		fileBuffer.WriteRune(c)

		switch c {
		case 'i':
			if prevChar == ' ' || prevChar == '\t' {
				//import的开始
				isImportChar = true
			} else {
				isImportChar = false
			}
		case 'm':
			if isImportChar && prevChar == 'i' {
				//import的m判断
				isImportChar = true
			} else {
				isImportChar = false
			}
		case 'p':
			if isImportChar && prevChar == 'm' {
				//import的p判断
				isImportChar = true
			} else {
				isImportChar = false
			}
		case 'o':
			if isImportChar && prevChar == 'p' {
				//import的o判断
				isImportChar = true
			} else {
				isImportChar = false
			}
		case 'r':
			if isImportChar && prevChar == 'o' {
				//import的r判断
				isImportChar = true
			} else {
				isImportChar = false
			}
		case 't':
			if isImportChar && prevChar == 'r' {
				//import的t判断
				isImportChar = true
			} else {
				isImportChar = false
			}
		case ' ':
			if isImportChar && prevChar == 't' {
				//import的判断
				isImportString = true
			}
			isImportChar = false
		case '\t':
			if isImportChar && prevChar == 't' {
				//import的判断
				isImportString = true
			}
			isImportChar = false
		case '(':
			if isImportChar && prevChar == 't' {
				//import的判断
				isImportString = true
			}
			if isImportString {
				//import的判断
				fileState.IsImportRegion = true
			}
			isImportChar = false
		case ')':
			if fileState.IsImportRegion {
				fileState.IsImportRegion = false
			}
		case '"':
			if fileState.IsImportRegion || isImportString {
				isQuoteRegion = true
				backLevel = 0
			} else {
				if isStringRegion {
					if prevChar != '\\' {
						isStringRegion = false
					}
				} else {	
					isStringRegion = true
				}
			}
		case '.':
			if isQuoteRegion {
				if prevChar == '"' || prevChar == '/' {
					dotCount = 1
				} else if prevChar == '.' {
					dotCount++
				} else {
					dotCount = 0
				}
			}
		case '/':
			if isQuoteRegion {
				if dotCount >= 2 {
					dotCount = 0
					backLevel++
				}
				break
			}
			if prevChar == '/' {
				isLineComment = true
				break
			}
		case '*':
			if prevChar == '/' {
				// 对块注释开始 "/*" 的判断
				fileState.IsBlockComment = true
			}
		case '`':
				// raw字符串符号
				fileState.IsRawStringRegion = true
		default:

		}

		prevChar = c
	}
}



func FixImportPath(importPath string, pathLevel int) {
	path := filepath.Clean(importPath)
	path = strings.Replace(path, "\\", "/", -1)
	
	tokens := strings.Split(path, "/")
	backCount := 0
	for i := range tokens {
		if tokens[i] == ".." {
			backCount++
		} else {
			break
		}
	}

	if backCount == 0 {
		if strings.HasPrefix(path, moduleName) {
			// 导入路径和模块名相同
			fileBuffer.WriteString(path)
		} else {
			//  !!!! 要先判断是否为
			//fileBuffer.WriteString(moduleName)
			if strings.HasPrefix(importPath, "./") {
				fileBuffer.WriteString(moduleName)
				fileBuffer.WriteRune('/')
				for i:=0; i<pathLevel; i++ {
					fileBuffer.WriteString(pathRel[i])
					fileBuffer.WriteRune('/')
				}
			}
			fileBuffer.WriteString(path)
		}
	} else {
		fileBuffer.WriteString(moduleName)

		forwardLevel := pathLevel - backCount
		
		for i:=0; i<forwardLevel; i++ {
			fileBuffer.WriteRune('/')
			fileBuffer.WriteString(pathRel[i])
		}
		
		count := len(tokens)
		for i:=backCount; i<count; i++ {
			fileBuffer.WriteRune('/')
			fileBuffer.WriteString(tokens[i])
		}

	}
	
}