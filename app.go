package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"text/template"

	"github.com/go-sql-driver/mysql"
)

var db *sql.DB

//validate templates
var tmpl = template.Must(template.ParseFiles("search.html", "results.html"))

// Album struct
type Album struct {
	ID     int64
	Title  string
	Artist string
	Price  float32
}

// Page structure
type Page struct {
	Title string
	Body  []Album
	Names []string
}

func main() {
	// Capture connection properties.
	cfg := mysql.Config{
		User:   os.Getenv("DBUSER"),
		Passwd: os.Getenv("DBPASS"),
		Net:    "tcp",
		Addr:   "127.0.0.1:3306",
		DBName: "recordings",
	}
	// Get a database handle.
	var err error
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("Connected!")

	// artist name here
	/* 	albums, err := albumsByArtist("John Coltrane")
	   	if err != nil {
	   		log.Fatal(err)
	   	}
	   	fmt.Printf("Albums found: %v\n", albums) */

	// GET user input here --for wuick testing w/put launch http server
	// fmt.Println("-> Enter artist name eg John Coltrane")
	// fmt.Println("-> Select a numeric option; \n [1] Book, Chapter & Verse - Dropdown Search \n [2] Keyword Search \n [3] ID Search")

	/* 	consoleReader := bufio.NewScanner(os.Stdin)
	   	consoleReader.Scan()
	   	artistFullname := consoleReader.Text()
	   	println("userChoice %s", artistFullname)

	   	albums, err := albumsByArtist(artistFullname)
	   	if err != nil {
	   		log.Fatal(err)
	   	}
	   	fmt.Printf("%s Albums found: %v\n", artistFullname, albums)
	*/

	//add some more http stuff:
	http.HandleFunc("/", searchHandler)
	http.HandleFunc("/results", resultsHandler)
	println("Serving http://localhost:8080")
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
	println("In resultsHandler")
	//convert price from string to float32?
	// stack overflow
	value, err := strconv.ParseFloat(r.FormValue("price"), 32)
	if err != nil {
		// do something sensible
	}
	price := float32(value)

	details := Album{
		Title:  r.FormValue("title"),
		Artist: r.FormValue("artist"),
		Price:  price,
	}

	//process details gathered for template execution
	// TODO: switch case 1,2,3 & default for various types of search ie artist(with dropdown), price(with range), title(dropdown)
	albums, err := albumsByArtist(details.Artist)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Albums found: %v\n", albums)

	// prep page data in page struct we created
	pageInfo := Page{
		Title: r.FormValue("title"),
		Body:  albums,
	}

	// parse & execute template
	tmpl, err = template.ParseFiles("results.html")
	if err != nil {
		log.Fatal(err) //TODO: add more to error log/why failed
	}
	tmpl.Execute(w, pageInfo)
}

// searchHandler - handle root page(search)
func searchHandler(w http.ResponseWriter, r *http.Request) {
	println("In searchHandler.")
	title := r.FormValue("title")
	artist := r.FormValue("artist")
	println("title: ", title)
	println("artist: ", artist)

	//TODO: get artist names from db and serve in clientside artists dropdown see below in "art"
	//something like:
	artists, err := allArtistNames()
	if err != nil {
		log.Fatal(err)
	}

	//print artists
	for _, a := range artists {
		print(a)
	}

	// check ?if index page sumbission not post then template is blank? i think it means
	if r.Method != http.MethodPost {
		tmpl.Execute(w, nil)
		return
	}

	// now serve artist names to frontend
	art := Page{
		Title: "WELCOME",
		Names: artists,
	}

	// parse & execute template
	tmpl, err = template.ParseFiles("search.html")
	if err != nil {
		log.Fatal(err) //TODO: add more to error log/why failed
	}
	tmpl.Execute(w, art)
}

// allArtistNames - helper func to get names of all artists in album table
func allArtistNames() ([]string, error) {
	// res us a slice to hold artist names returned
	var res []string

	// db query - distinct, no overlap
	rows, err := db.Query("SELECT DISTINCT artist from album;")
	if err != nil {
		return nil, fmt.Errorf("allArtistNames: %v", err)
	}

	defer rows.Close()
	//loop through rows, put names in slice of strings we created
	var alb string // temp string to store distinct artist names
	//if data in rows exists
	for rows.Next() {
		if err := rows.Scan(&alb); err != nil {
			return nil, fmt.Errorf("In allArtistNames: %v", err)
		}
		res = append(res, alb)
	}
	// if error in rows ie rows.Err()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("allArtistNames: %v", err)
	}
	return res, nil

}
