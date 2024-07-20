package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
)

var mu sync.Mutex

type Scanner struct {
	Dist float64
	X    float64
	Y    float64
	id   int
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
	fmt.Println("Data from file ", string(content))
	data := string(content)
	lines := strings.Split(data, "\n")
	var scanners []Scanner
	scannerMap := make(map[string]Scanner)
	for _, line := range lines {
		if line == "" {
			continue
		}
		var scanner Scanner
		_, err := fmt.Sscanf(line, "%f %f %f %d", &scanner.Dist, &scanner.X, &scanner.Y, &scanner.id)
		if err != nil {
			fmt.Println("Error parsing line:", err)
			return nil, err
		}

		key := fmt.Sprintf("%f,%f,%d", scanner.X, scanner.Y, scanner.id)
		scannerMap[key] = scanner
	}

	for _, scanner := range scannerMap {
		scanners = append(scanners, scanner)
	}

	sort.Slice(scanners, func(i, j int) bool {
		if scanners[i].id == scanners[j].id {
			return scanners[i].Dist > scanners[j].Dist
		}
		return scanners[i].id < scanners[j].id
	})

	var calculation []Scanner
	var indexes []int
	var scanStamp Scanner
	compared := 0
	added := false
	guessedId := 1
	var toCalculate []Scanner
	if len(scanners) > 0 {
		for _, scanner := range scanners {
			if scanner.id == guessedId {
				compared += 1
				toCalculate = append(toCalculate, scanner)
				if compared >= 3 {
					for i := 0; i < 3; i++ {
						dist := toCalculate[i].Dist
						dist = math.Pow(10, ((-70 - dist) / (10 * 2)))
						scanStamp.Dist = dist
						scanStamp.X = toCalculate[i].X
						scanStamp.Y = toCalculate[i].Y
						scanStamp.id = guessedId
						calculation = append(calculation, scanStamp)
					}
					var x float64
					var y float64
					if calculation[0].X == calculation[1].X {
						x_podstav := (math.Pow(calculation[1].Dist, 2) - math.Pow(calculation[1].Y, 2) -
							math.Pow(calculation[0].Dist, 2) + math.Pow(calculation[0].Y, 2) + math.Pow(calculation[0].X, 2) -
							math.Pow(calculation[1].X, 2)) / (2 * (calculation[0].Y - calculation[1].Y))
						koef := (calculation[1].X - calculation[0].X) / (calculation[0].Y - calculation[1].Y)

						x = (math.Pow(calculation[2].Dist, 2) - math.Pow(calculation[2].Y, 2) +
							(2 * calculation[2].Y * x_podstav) - math.Pow(calculation[0].Dist, 2) +
							math.Pow(calculation[0].Y, 2) - (2 * calculation[0].Y * x_podstav) +
							math.Pow(calculation[0].X, 2) - math.Pow(calculation[2].X, 2)) /
							(2*calculation[0].Y*koef + 2*calculation[0].X -
								2*calculation[2].Y*koef - 2*calculation[2].X)
						y = x_podstav + koef*x
					} else {
						x_podstav := (math.Pow(calculation[1].Dist, 2) - math.Pow(calculation[1].X, 2) -
							math.Pow(calculation[0].Dist, 2) + math.Pow(calculation[0].X, 2) + math.Pow(calculation[0].Y, 2) -
							math.Pow(calculation[1].Y, 2)) / (2 * (calculation[0].X - calculation[1].X))
						koef := (calculation[1].Y - calculation[0].Y) / (calculation[0].X - calculation[1].X)

						y = (math.Pow(calculation[2].Dist, 2) - math.Pow(calculation[2].X, 2) +
							(2 * calculation[2].X * x_podstav) - math.Pow(calculation[0].Dist, 2) +
							math.Pow(calculation[0].X, 2) - (2 * calculation[0].X * x_podstav) +
							math.Pow(calculation[0].Y, 2) - math.Pow(calculation[2].Y, 2)) /
							(2*calculation[0].X*koef + 2*calculation[0].Y -
								2*calculation[2].X*koef - 2*calculation[2].Y)
						x = x_podstav + koef*y
					}
					fmt.Println("Coordinates: ", x, ";", y, "for id: ", guessedId)
					options := &redis.Options{
						Addr:     "77.221.156.184:6379", // Change this to your Redis server address
						Password: "",                    // No password by default
						DB:       0,                     // Use default DB
					}
					client := redis.NewClient(options)

					currentTime := time.Now().UnixNano() / int64(time.Millisecond)
					size, err := client.DBSize(context.Background()).Result()
					if err != nil {
						fmt.Println("Error getting database size:", err)
					}
					size += 1

					playerID := guessedId
					playerX := x
					playerY := y
					redisKey := strconv.FormatInt(size, 10)
					playerDataRedis := map[string]interface{}{
						"id":        strconv.Itoa(playerID),
						"x":         fmt.Sprintf("%.15f", playerX),
						"y":         fmt.Sprintf("%.15f", playerY),
						"timestamp": strconv.FormatInt(currentTime, 10),
					}
					err = client.HMSet(context.Background(), redisKey, playerDataRedis).Err()
					if err != nil {
						fmt.Println("Error setting hash field-value pairs:", err)
					}
					fmt.Printf("Successfully set hash field-value pairs in Redis under key '%s' with values %f %f %d\n", redisKey, playerX, playerY, playerID)
					client.Close()

					added = true
					indexes = append(indexes, guessedId)
					calculation = calculation[:0]
					toCalculate = toCalculate[:0]
					compared = 0
				}
			} else {
				compared = 1
				toCalculate = toCalculate[:0]
				toCalculate = append(toCalculate, scanner)
				guessedId = scanner.id
			}
		}
	}
	if added {
		file, err := os.OpenFile("cur.txt", os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			fmt.Println("Error opening file:", err)
		}
		err = file.Truncate(0)
		if err != nil {
			fmt.Println("Error truncating file:", err)
		}
		fmt.Println("File content has been erased.")
		file.Close()
		file, err = os.OpenFile("cur.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println("Error opening file:", err)
		}
		for _, fetch := range scanners {
			check := true
			for _, ids := range indexes {
				if fetch.id == ids {
					check = false
				}
			}
			if check {
				if _, err := file.WriteString(fmt.Sprintf("%f %f %f %d\n", fetch.Dist, fetch.X, fetch.Y, fetch.id)); err != nil {
					fmt.Println("Error writing to file ", err)
				}
			}
		}
		file.Close()
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
	fmt.Println(data)
	value1, ok1 := data["key1"].(float64)
	value2, ok2 := data["key2"].(float64)
	value3, ok3 := data["key3"].(float64)
	value4, ok4 := data["key4"].(float64)

	if !ok1 || !ok2 || !ok3 || !ok4 {
		fmt.Println("Invalid JSON data format")
	}

	fmt.Printf("Value1: %f, Value2: %f, Value3: %f, Value4: %f\n", value1, value2, value3, value4)

	file, err := os.OpenFile("cur.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return err
	}
	defer file.Close()
	val := int(value4)
	// Write data to the file
	if _, err := file.WriteString(fmt.Sprintf("%f %f %f %d\n", value1, value2, value3, val)); err != nil {
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
