package main

import (
	"fmt"
	"io/ioutil"
	"os"
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

	data := string(content)
	lines := strings.Split(data, "\n")
	var scanners []Scanner
	for _, line := range lines {
		var scanner Scanner
		_, err := fmt.Sscanf(line, "%f %f %f", &scanner.Dist, &scanner.X, &scanner.Y)
		if err != nil {
			fmt.Println("Error parsing line:", err)
			return nil, err
		}

		scanners = append(scanners, scanner)
	}
	return scanners, nil
}

func writeToFile(body string) error {
	mu.Lock()
	defer mu.Unlock()

	file, err := os.OpenFile("cur.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return err
	}
	defer file.Close()

	// Write data to the file
	if _, err := file.WriteString(body + "\n"); err != nil {
		fmt.Println("Error writing to file:", err)
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

		err := writeToFile(string(body))
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
