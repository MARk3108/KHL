package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

var mu sync.Mutex

type Scanner struct {
	Dist float64
	X    float64
	Y    float64
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = ":3000"
	} else {
		port = ":" + port
	}

	return port
}

func readFromFile() ([]Scanner, error) {
	mu.Lock()
	defer mu.Unlock()

	content, err := ioutil.ReadFile("cur.txt")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return nil, err
	}
	fmt.Println("Data from file ", content)
	data := string(content)
	lines := strings.Split(data, "\n")
	var scanners []Scanner
	scannerMap := make(map[string]Scanner)
	for _, line := range lines {
		if line == "" {
			continue
		}
		var scanner Scanner
		_, err := fmt.Sscanf(line, "%f %f %f", &scanner.Dist, &scanner.X, &scanner.Y)
		if err != nil {
			fmt.Println("Error parsing line:", err)
			return nil, err
		}

		key := fmt.Sprintf("%f,%f", scanner.X, scanner.Y)
		scannerMap[key] = scanner
	}

	for _, scanner := range scannerMap {
		scanners = append(scanners, scanner)
	}

	sort.Slice(scanners, func(i, j int) bool {
		return scanners[i].Dist > scanners[j].Dist
	})

	var calculation []Scanner
	var scanStamp Scanner

	if len(scanners) >= 3 {
		for i := 0; i < 3; i++ {
			dist := scanners[i].Dist
			dist = math.Pow(10, ((-84 - dist) / (10 * 2)))
			scanStamp.Dist = dist
			scanStamp.X = scanners[i].X
			scanStamp.Y = scanners[i].Y
			calculation = append(calculation, scanStamp)
		}
		x_podstav := (math.Pow(calculation[1].Dist, 2) - math.Pow(calculation[1].X, 2) -
			math.Pow(calculation[0].Dist, 2) + math.Pow(calculation[0].X, 2) + math.Pow(calculation[0].Y, 2) -
			math.Pow(calculation[1].Y, 2)) / (2 * (calculation[0].X - calculation[1].X))
		koef := (calculation[1].Y - calculation[0].Y) / (calculation[0].X - calculation[1].X)

		y := (math.Pow(calculation[2].Dist, 2) - math.Pow(calculation[2].X, 2) +
			(2 * calculation[2].X * x_podstav) - math.Pow(calculation[0].Dist, 2) +
			math.Pow(calculation[0].X, 2) - (2 * calculation[0].X * x_podstav) +
			math.Pow(calculation[0].Y, 2) - math.Pow(calculation[2].Y, 2)) /
			(2*calculation[0].X*koef + 2*calculation[0].Y -
				2*calculation[2].X*koef - 2*calculation[2].Y)
		x := x_podstav + koef*y
		fmt.Println("Coordinates: ", x, ";", y)

		file, err := os.OpenFile("cur.txt", os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			fmt.Println("Error opening file:", err)
		}
		defer file.Close()

		// Обрезать файл до нулевой длины (удалить все данные)
		err = file.Truncate(0)
		if err != nil {
			fmt.Println("Error truncating file:", err)
		}
		fmt.Println("File content has been erased.")
	}

	return scanners, nil
}

func writeToFile(body []byte) error {
	mu.Lock()
	defer mu.Unlock()
	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println("Error parsing JSON data: ", err)
	}
	value1, ok1 := data["key1"].(float64)
	value2, ok2 := data["key2"].(float64)
	value3, ok3 := data["key3"].(float64)

	if !ok1 || !ok2 || !ok3 {
		fmt.Println("Invalid JSON data format")
	}

	fmt.Printf("Value1: %f, Value2: %f, Value3: %f\n", value1, value2, value3)

	file, err := os.OpenFile("cur.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return err
	}
	defer file.Close()

	// Write data to the file
	if _, err := file.WriteString(fmt.Sprintf("%f %f %f\n", value1, value2, value3)); err != nil {
		fmt.Println("Error writing to file ", err)
		return err
	}

	return nil
}

func main() {
	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Main page",
		})
	})

	app.Post("/write", func(c *fiber.Ctx) error {
		body := c.Body()

		// Выводим тело запроса в консоль
		fmt.Println("Received request body:", body)

		err := writeToFile(body)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Error writing to file")
		}

		// Return a success message to the client
		return c.SendString("Data received and written to file successfully")
	})

	app.Get("/read", func(c *fiber.Ctx) error {
		content, err := ioutil.ReadFile("cur.txt")
		if err != nil {
			fmt.Println("Error reading file:", err)
		}
		return c.JSON(fiber.Map{"massage": string(content)})
	})

	go func() {
		for {
			content, err := readFromFile()
			if err != nil {
				fmt.Println("Error reading from file:", err)
			} else {
				fmt.Println("File content:", content)
			}

			// Sleep for one second
			time.Sleep(1 * time.Second)
		}
	}()

	app.Listen(getPort())
}
