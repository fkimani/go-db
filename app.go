package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
)

var db *sql.DB

// Album struct
type Album struct {
	ID     int64
	Title  string
	Artist string
	Price  float32
}

// // Page structure
// type Page struct {
// 	Title string
// 	Body  []byte
// }

func searchHandler(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	artist := r.FormValue("artist")

	println("title: ", title)
	println("artist: ", artist)
}
func main() {
	// Capture connection properties.
	// cfg := mysql.Config{
	// 	User:   os.Getenv("DBUSER"),
	// 	Passwd: os.Getenv("DBPASS"),
	// 	Net:    "tcp",
	// 	Addr:   "127.0.0.1:3306",
	// 	DBName: "recordings",
	// }
	// // Get a database handle.
	// var err error
	// db, err = sql.Open("mysql", cfg.FormatDSN())
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// pingErr := db.Ping()
	// if pingErr != nil {
	// 	log.Fatal(pingErr)
	// }
	// fmt.Println("Connected!")

	// // artist name here
	// /* 	albums, err := albumsByArtist("John Coltrane")
	//    	if err != nil {
	//    		log.Fatal(err)
	//    	}
	//    	fmt.Printf("Albums found: %v\n", albums) */

	// // GET user input here
	// fmt.Println("-> Enter artist name eg John Coltrane")
	// // fmt.Println("-> Select a numeric option; \n [1] Book, Chapter & Verse - Dropdown Search \n [2] Keyword Search \n [3] ID Search")

	// consoleReader := bufio.NewScanner(os.Stdin)
	// consoleReader.Scan()
	// artistFullname := consoleReader.Text()
	// println("userChoice %s", artistFullname)

	// albums, err := albumsByArtist(artistFullname)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Printf("%s Albums found: %v\n", artistFullname, albums)

	//add some more http stuff:
	http.HandleFunc("/", searchHandler)
	http.HandleFunc("/results", resultsHandler)
	println("Serving http://localhost/8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

	// Hard-code ID 2 here to test the query.
	/* alb, err := albumByID(2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Album found: %v\n", alb) */

	// add album
	/* albID, err := addAlbum(Album{
		Title:  "The Modern Sound of Betty Carter",
		Artist: "Betty Carter",
		Price:  49.99,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("ID of added album: %v\n", albID)*/
}

// albumsByArtist queries for albums that have the specified artist name.
func albumsByArtist(name string) ([]Album, error) {
	// An albums slice to hold data from returned rows.
	var albums []Album

	rows, err := db.Query("SELECT * FROM album WHERE artist = ?", name)
	if err != nil {
		return nil, fmt.Errorf("albumsByArtist %q: %v", name, err)
	}
	defer rows.Close()
	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var alb Album
		if err := rows.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
			return nil, fmt.Errorf("albumsByArtist %q: %v", name, err)
		}
		albums = append(albums, alb)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("albumsByArtist %q: %v", name, err)
	}
	return albums, nil
}

// albumByID queries for the album with the specified ID.
func albumByID(id int64) (Album, error) {
	// An album to hold data from the returned row.
	var alb Album

	row := db.QueryRow("SELECT * FROM album WHERE id = ?", id)
	if err := row.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
		if err == sql.ErrNoRows {
			return alb, fmt.Errorf("albumsById %d: no such album", id)
		}
		return alb, fmt.Errorf("albumsById %d: %v", id, err)
	}
	return alb, nil
}

// addAlbum adds the specified album to the database,
// returning the album ID of the new entry
func addAlbum(alb Album) (int64, error) {
	result, err := db.Exec("INSERT INTO album (title, artist, price) VALUES (?, ?, ?)", alb.Title, alb.Artist, alb.Price)
	if err != nil {
		return 0, fmt.Errorf("addAlbum: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("addAlbum: %v", err)
	}
	return id, nil

}

// http functions
func resultsHandler(w http.ResponseWriter, r *http.Request) {
	artist := "the artist"
	println("artist ", artist)
	fmt.Fprintf(w, "<h1>some heading</h1><div>%s</div>", artist)
}
