package utils

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/d-iii-s/slsbench/internal/model"
	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
	"io"
	"io/fs"
	"log"
	"net"
	"os"
	"path/filepath"
)

const (
	// Name hints
	HintFirstName = "firstName"
	HintLastName  = "lastName"
	HintFullName  = "fullName"
	HintName      = "name"
	
	// Internet hints
	HintEmail      = "email"
	HintUsername   = "username"
	HintPassword   = "password"
	HintURL        = "url"
	HintURI        = "uri"
	HintDomainName = "domainName"
	HintHostname   = "hostname"
	HintIP         = "ip"
	HintIPv4       = "ipv4"
	HintIPv6       = "ipv6"
	
	// Address hints
	HintCity    = "city"
	HintState   = "state"
	HintCountry = "country"
	
	// String hints
	HintWord   = "word"
	HintString = "string"
	
	// Date hints
	HintDate     = "date"
	HintTimestamp = "timestamp"
	HintDateTime = "dateTime"
	HintISO8601  = "iso8601"
	
	// Datatype hints
	HintUUID    = "uuid"
	HintNumber  = "number"
	HintInteger = "integer"
	HintInt     = "int"
	HintFloat   = "float"
	HintDouble  = "double"
	HintBoolean = "boolean"
	HintByte    = "byte"
	HintBinary  = "binary"
	
	// ID hint
	HintID = "id"
)

const (
	DefaultWorkloadGeneratorImage = "aape2k/workload-generator-sessions:latest"
)

var HintOptions = []string{
	// Name
	HintFirstName,
	HintLastName,
	HintFullName,
	HintName,
	
	// Internet
	HintEmail,
	HintUsername,
	HintPassword,
	HintURL,
	HintURI,
	HintDomainName,
	HintHostname,
	HintIP,
	HintIPv4,
	HintIPv6,
	
	// Address
	HintCity,
	HintState,
	HintCountry,
	
	// String
	HintWord,
	HintString,
	
	// Date
	HintDate,
	HintTimestamp,
	HintDateTime,
	HintISO8601,
	
	// Datatype
	HintUUID,
	HintNumber,
	HintInteger,
	HintInt,
	HintFloat,
	HintDouble,
	HintBoolean,
	HintByte,
	HintBinary,
	
	// ID
	HintID,
}

//func loadEndpointInfoFromJson(filename string) (*model.QueryInfo, error) {
//	bytes, err := os.ReadFile(filename)
//	if err != nil {
//		return nil, err
//	}
//
//	var data *model.QueryInfo
//	err = json.Unmarshal(bytes, &data)
//	if err != nil {
//		return nil, err
//	}
//
//	data.Endpoint = "http://%s:%s" + data.Endpoint
//
//	return data, nil
//}

// Copy starts with capital letter to avoid name collision with built-in function
func Copy(src, dst, dstFileName string) error {
	info, err := os.Stat(src)
	if err != nil {
		log.Panicf("stat source: %w", err)
	}

	if info.IsDir() {
		return CopyDir(src, dst, nil)
	}
	dst = filepath.Join(dst, dstFileName)
	return CopyFile(src, dst)
}

func OpenOpenApiSpecFile(ctx context.Context, path string) *openapi3.T {
	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromFile(path)
	if err != nil {
		log.Fatalf("failed to load specification: %v", err)
	}

	// invalid memory address every time ?
	//if err := spec.Validate(nil); err != nil {
	//	log.Printf("warning: spec not valid: %v", err)
	//}

	spec.InternalizeRefs(ctx, nil)

	return spec
}

// CopyFilePreserveName copies the file from src to the dst directory without changing its name.
func CopyFilePreserveName(src, dstDir string) error {
	// Ensure the destination directory exists
	if err := os.MkdirAll(dstDir, os.ModePerm); err != nil {
		return err
	}

	// Get the base filename from the source path
	filename := filepath.Base(src)

	// Create full destination path
	dst := filepath.Join(dstDir, filename)

	return CopyFile(src, dst)
}

func CopyFile(srcFile, dstFile string) error {
	src, err := os.Open(srcFile)
	if err != nil {
		log.Panicf("open source file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(dstFile)
	if err != nil {
		log.Panicf("create destination file: %w", err)
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		log.Panicf("copy contents: %w", err)
	}

	return nil
}

func CopyDir(srcDir, dstDir string, exclude *[]string) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if exclude != nil && contains(*exclude, d.Name()) {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dstDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, os.ModePerm)
		}

		return CopyFile(path, targetPath)
	})
}

func SaveYAML(path string, data any) error {
	outBytes, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, outBytes, 0o644); err != nil {
		return err
	}
	log.Println("\nDone. Updated spec written to", path)
	return nil
}

func isPortAvailable(port string) bool {
	address := fmt.Sprintf("localhost:%s", port)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		log.Printf("Required port %s is not available. Error connecting to %s: %s", port, address, err)
		return false // Port is in use or blocked
	}
	ln.Close() // Close the listener so others can use the port
	return true
}

func parseJSONFile[T any](path string, out *T) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(out); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}
	return nil
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func fileSHA256(path string) string {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("Error opening file for hash calculation %s: %s", path, err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		log.Fatalf("Error hashing file %s: %s", path, err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
}

// FindByName searches for an ApiConfig by name in the list.
// Returns a pointer to the ApiConfig if found, otherwise nil.
//func (list model.ApiConfigList) FindByName(name string) *model.ApiConfig {
//	for i := range list {
//		if list[i].Name == name {
//			return &list[i]
//		}
//	}
//	return nil
//}

func SaveData(snapshots []model.Snapshot, path string) {
	file, err := os.Create(path)
	if err != nil {
		fmt.Println("Error creating JSON file:", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty print JSON
	if err := encoder.Encode(snapshots); err != nil {
		fmt.Println("Error encoding JSON:", err)
	}
}

func WriteCSV(filePath string, header []string, rows [][]string) error {
	// Create or truncate the file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("could not create file: %v", err)
	}
	defer file.Close()

	// Create CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("could not write header: %v", err)
	}

	// Write rows
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("could not write row: %v", err)
		}
	}

	return nil
}

// DumpJSON returns a JSON-formatted string representation of any struct
func DumpJSON(v any) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("error marshaling to JSON: %v", err)
	}
	return string(data)
}

// PrintJSON prints any struct in a readable JSON format
func PrintJSON(v any) {
	fmt.Println(DumpJSON(v))
}
