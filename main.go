package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/gofiber/fiber/v2"
)

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = ":3000"
	} else {
		port = ":" + port
	}

	return port
}

func main() {
	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Hello, Railway!",
		})
	})

	app.Post("/write", func(c *fiber.Ctx) error {
		body := c.Body()

		// Выводим тело запроса в консоль
		fmt.Println("Received request body:", body)

		file, err := os.OpenFile("cur.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println("Error opening file:", err)
		}
		defer file.Close()

		// Записываем данные в файл
		if _, err := file.WriteString(string(body) + "\n"); err != nil {
			fmt.Println("Error writing to file:", err)
		}

		// Возвращаем сообщение клиенту
		return c.SendString("Data received successfully")
	})

	app.Get("/read", func(c *fiber.Ctx) error {
		content, err := ioutil.ReadFile("cur.txt")
		if err != nil {
			fmt.Println("Error reading file:", err)
		}
		return c.JSON(fiber.Map{"massage": content})
	})
	app.Listen(getPort())
}
