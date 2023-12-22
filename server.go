package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	gon "github.com/rafiathallah3/Gon"
)

type Pool struct {
	Register   chan *Client
	Unregister chan *Client
	Clients    map[*Client]bool
	Broadcast  chan Pesan
}

func (pool *Pool) Start() {
	for {
		select {
		case client := <-pool.Register:
			pool.Clients[client] = true
			fmt.Println("Size of Connection Pool: ", len(pool.Clients), " JOINED")

			for client := range pool.Clients {
				client.Conn.WriteJSON(Pesan{Tipe: 1, Isi: "New User Joined...", Online: len(pool.Clients)})
			}

			break
		case client := <-pool.Unregister:
			delete(pool.Clients, client)
			fmt.Println("Size of Connection Pool: ", len(pool.Clients), " DISCONNECT!")

			fmt.Println("KELUAR", client.DalamKamar)
			if client.DalamKamar != nil {
				kamar_dipilih := client.DalamKamar
				client.DalamKamar = nil
				kamar_dipilih.Pemain_2.DalamKamar = nil

				fmt.Println("ADA KELUAR!")
				kamar_dipilih.Pemain_1.Conn.WriteJSON(Pesan{Tipe: 2, Isi: fmt.Sprintf(`{ "info": "Anonymous keluar dari room!", "id": "%s"}`, "")})
				kamar_dipilih.Pemain_2.Conn.WriteJSON(Pesan{Tipe: 2, Isi: fmt.Sprintf(`{ "info": "Anonymous keluar dari room!", "id": "%s"}`, "")})

				delete(SemuaKamar, kamar_dipilih)
			}

			delete(PencarianPemain, client)

			for client := range pool.Clients {
				client.Conn.WriteJSON(Pesan{Tipe: 1, Isi: "User Disconnected...", Online: len(pool.Clients)})
			}
			break
		case message := <-pool.Broadcast:
			fmt.Println("Sending message to all clients in Pool")

			var PesanClient Pesan
			if err := json.Unmarshal([]byte(message.Isi), &PesanClient); err != nil {
				fmt.Println("ERROR DISINI", message.Isi)
				fmt.Println(err)
				return
			}

			if PesanClient.Tipe == 1 && PesanClient.Isi == "Cari" {
				PencarianPemain[message.Dari] = true

				for client := range PencarianPemain {
					if client != message.Dari && client.DalamKamar == nil {
						kamar := &Kamar{Pemain_1: message.Dari, Pemain_2: client}
						SemuaKamar[kamar] = true
						message.Dari.DalamKamar = kamar
						client.DalamKamar = kamar

						message.Dari.ID_Akun = 1
						if err := message.Dari.Conn.WriteJSON(Pesan{Tipe: 2, Isi: fmt.Sprintf(`{ "info": "You are now in a room!", "id": %d}`, 1)}); err != nil {
							fmt.Println(err)
							return
						}
						delete(PencarianPemain, message.Dari)

						client.ID_Akun = 2
						if err := client.Conn.WriteJSON(Pesan{Tipe: 2, Isi: fmt.Sprintf(`{ "info": "You are now in a room!", "id": %d}`, 2)}); err != nil {
							fmt.Println(err)
							return
						}
						delete(PencarianPemain, client)

						fmt.Println("Masuk Kamar")
					}
				}

				fmt.Println("Total Cari: ", len(PencarianPemain))
			}

			if PesanClient.Tipe == 1 && PesanClient.Isi == "Stop" {
				fmt.Println("Hapus Kamar")
				if message.Dari.DalamKamar != nil {
					kamar_dipilih := message.Dari.DalamKamar
					message.Dari.DalamKamar = nil
					kamar_dipilih.Pemain_2.DalamKamar = nil

					fmt.Println("ADA KELUAR!")
					kamar_dipilih.Pemain_1.Conn.WriteJSON(Pesan{Tipe: 2, Isi: fmt.Sprintf(`{ "info": "Someone left the room!", "id": "%s"}`, "")})
					kamar_dipilih.Pemain_2.Conn.WriteJSON(Pesan{Tipe: 2, Isi: fmt.Sprintf(`{ "info": "Someone left the room!", "id": "%s"}`, "")})

					delete(SemuaKamar, kamar_dipilih)
				}

				delete(PencarianPemain, message.Dari)
			}

			fmt.Println(PesanClient.Tipe, message.Dari.DalamKamar)
			if PesanClient.Tipe == 2 && message.Dari.DalamKamar != nil {
				fmt.Println("kirim pesan di room!")
				DataPesan := Pesan{Tipe: 3, Isi: fmt.Sprintf(`{ "info": "%s", "id": %d}`, PesanClient.Isi, message.Dari.ID_Akun)}
				if err := message.Dari.DalamKamar.Pemain_1.Conn.WriteJSON(DataPesan); err != nil {
					fmt.Println(err)
					return
				}

				if err := message.Dari.DalamKamar.Pemain_2.Conn.WriteJSON(DataPesan); err != nil {
					fmt.Println(err)
					return
				}
			}

			// 3, yang artinya mengirim offer video
			if PesanClient.Tipe == 3 {
				KirimKeKlient := &Client{}

				if message.Dari.DalamKamar.Pemain_1.ID_Akun == 2 {
					KirimKeKlient = message.Dari.DalamKamar.Pemain_1
				}

				if message.Dari.DalamKamar.Pemain_2.ID_Akun == 2 {
					KirimKeKlient = message.Dari.DalamKamar.Pemain_2
				}

				fmt.Println("Client", KirimKeKlient)
				if err := KirimKeKlient.Conn.WriteJSON(Pesan{Tipe: 4, Isi: PesanClient.Isi}); err != nil {
					fmt.Println(err)
					return
				}
			}

			//4, yang artinya mendapatkan jawaban video
			if PesanClient.Tipe == 4 {
				KirimKeKlient := &Client{}

				if message.Dari.DalamKamar.Pemain_1.ID_Akun == 1 {
					KirimKeKlient = message.Dari.DalamKamar.Pemain_1
				}

				if message.Dari.DalamKamar.Pemain_2.ID_Akun == 1 {
					KirimKeKlient = message.Dari.DalamKamar.Pemain_2
				}

				fmt.Println("Client", KirimKeKlient)
				if err := KirimKeKlient.Conn.WriteJSON(Pesan{Tipe: 5, Isi: PesanClient.Isi}); err != nil {
					fmt.Println(err)
					return
				}
			}

			//5, yang artinya mengirim Ice candidate
			if PesanClient.Tipe == 5 {
				KirimKeKlient := &Client{}

				if message.Dari.ID_Akun != message.Dari.DalamKamar.Pemain_1.ID_Akun {
					KirimKeKlient = message.Dari.DalamKamar.Pemain_1
				}

				if message.Dari.ID_Akun != message.Dari.DalamKamar.Pemain_2.ID_Akun {
					KirimKeKlient = message.Dari.DalamKamar.Pemain_2
				}

				if err := KirimKeKlient.Conn.WriteJSON(Pesan{Tipe: 6, Isi: PesanClient.Isi}); err != nil {
					fmt.Println(err)
					return
				}
			}
		}
	}
}

type Kamar struct {
	Pemain_1 *Client
	Pemain_2 *Client
}

type Client struct {
	Conn       *websocket.Conn
	Pool       *Pool
	DalamKamar *Kamar
	ID_Akun    int
}

func (c *Client) Baca() {
	defer func() {
		c.Pool.Unregister <- c
		c.Conn.Close()
	}()

	for {
		messageType, p, err := c.Conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}

		message := Pesan{Dari: c, Tipe: messageType, Isi: string(p), Online: len(c.Pool.Clients)}
		c.Pool.Broadcast <- message
		// fmt.Printf("Message Received: %+v\n", message)
	}
}

type Pesan struct {
	Dari   *Client
	Tipe   int    `json:"tipe"`
	Isi    string `json:"isi"`
	Online int    `json:"online"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1204,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var PencarianPemain = make(map[*Client]bool)
var SemuaKamar = make(map[*Kamar]bool)

func NewPool() *Pool {
	return &Pool{
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan Pesan),
	}
}

func main() {
	server := gon.New()
	pool := NewPool()
	go pool.Start()

	server.SetFuncMap(gon.FuncMap{
		"getLength": func(s string) int {
			return len(s)
		},
	})

	server.Route(gon.GET, "/", func(context *gon.Context) {
		context.Render_template("index.html", "base", gon.TempVar{
			"judul": "Horizon Talk",
		})
	})

	server.Route(gon.GET, "/ws", func(ctx *gon.Context) {
		ws, err := upgrader.Upgrade(ctx.Response, ctx.Request, nil)
		if err != nil {
			fmt.Println(err)
			ctx.Error("Tidak bisa connect socket", 400)
		}

		client := &Client{
			Conn: ws,
			Pool: pool,
		}

		pool.Register <- client
		client.Baca()
		// serveWs(hub, ctx.Response, ctx.Request)
	})

	server.SetIcon("./favicon.ico")

	server.Run(":8000")
}
