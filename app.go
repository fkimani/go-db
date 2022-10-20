package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"text/template"

	"github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

var db *sql.DB

// validate templates
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
	// Titles map[string]interface{}
	Titles []string
	Body   []Album
	Names  []string
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

	l := log.WithFields(log.Fields{
		"Action":  "Main",
		"DB":      db,
		"DB Name": cfg.DBName,
	})
	fmt.Println("Connected!")
	l.Info()
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
	//just TESTS
	/* fmt.Println("TEST IN MAIN... ")
	fmt.Print("Album names: ")
	fmt.Println(allAlbumNames())
	fmt.Println("Album by:*** ")
	fmt.Println(albumsByArtist("John Coltrane"))
	fmt.Println("***")
	fmt.Println(albumByID(1))
	fmt.Println(allArtistNames())
	a, _ := queryData("Jeru")
	fmt.Println("a: ", a) */

	//http calls:
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

// NEW
// searchHandler - handler for search
func searchHandler(w http.ResponseWriter, r *http.Request) {
	println("*In searchHandler*")

	//fetch dropdown for artists
	artistsList, err := allArtistNames()
	if err != nil {
		log.Fatal(err)
	}
	//fetch dropdown for album titles
	titlesList, err := allAlbumNames()
	if err != nil {
		log.Fatal(err)
	}

	//DB orig query artists albums
	/* 	albums, err := albumsByArtist(r.FormValue("artist"))
	   	if err != nil {
	   		log.Fatal(err)
	   	}
	   	fmt.Println("selectedTitle Albums found: %v", albums) */

	//TODO
	// $ cat switch-example.md for example

	/* 	//DB query (move this to switch option 2)---MOVE to RESULTS HANDLER**
	   	artistFullname := r.FormValue("artist")
	   	println("artistFullname? ", artistFullname)
	   	albumBy, err := albumsByArtist(artistFullname)
	   	if err != nil {
	   		log.Fatal(err)
	   	}
	   	fmt.Println("Albums found: %v", albumBy) */

	// prepare page struct for title and artist dropdowns
	art := Page{
		Titles: titlesList,
		Names:  artistsList,
		// Body:   albumBy,
	}

	// parse & execute template
	tmpl, err = template.ParseFiles("search.html")
	if err != nil {
		log.Fatalf("Search Handler ParseFiles Error: %v", err) //TODO: add more to error log/why failed
	}
	tmpl.Execute(w, art)
}

// resultHandler - handler for results
func resultsHandler(w http.ResponseWriter, r *http.Request) {
	println("In resultsHandler**")
	//convert price from string to float32?
	// stack overflow
	value, err := strconv.ParseFloat(r.FormValue("price"), 32)
	if err != nil {
		// do something sensible
	}
	price := float32(value)

	println("rH Title: ", r.FormValue("title"))
	println("rH Artist: ", r.FormValue("artist"))
	//TODO: get artist and title from select option using "selected" html/tempalte action
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
	fmt.Println("%T's Albums found: %T", details.Artist, albums)
	// for _, a := range albums {
	// 	println(a)
	// }

	// prep page data in page struct we created
	pageInfo := Page{
		// Title: r.FormValue("title"),
		Body: albums,
	}

	// parse & execute template
	tmpl, err = template.ParseFiles("results.html")
	if err != nil {
		log.Fatal(err) //TODO: add more to error log/why failed
	}
	tmpl.Execute(w, pageInfo)
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
	//print columns
	col, _ := rows.Columns()
	fmt.Println("Print columns:", col[0])

	defer rows.Close()
	//loop through rows, put names in slice of strings we created
	var alb string // temp string to store distinct artist names
	//if data in rows exists
	for rows.Next() {
		if err := rows.Scan(&alb); err != nil {
			return nil, fmt.Errorf("In allArtistNames: %v", err)
		}
		// res = append(res, alb)
		// res = append(res, fmt.Sprintf("%v", alb))
		res = append(res, fmt.Sprintf(alb))
	}
	// if error in rows ie rows.Err()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("allArtistNames: %v", err)
	}
	return res, nil
}

// albumNames
func allAlbumNames() ([]string, error) {
	var res []string

	// db query - distinct, no overlap
	cmd := "SELECT title from album;"
	rows, err := db.Query(cmd)
	if err != nil {
		return nil, fmt.Errorf("allAlbumNames: %v", err)
	}
	//print columns
	col, _ := rows.Columns()
	fmt.Println("Print columns:", col[0])

	defer rows.Close()
	//loop through rows, put names in slice of strings we created
	var alb string // temp string to store distinct artist names
	//if data in rows exists
	for rows.Next() {
		if err := rows.Scan(&alb); err != nil {
			return nil, fmt.Errorf("In allAlbumNames: %v", err)
		}
		// res = append(res, alb)
		// res = append(res, fmt.Sprintf("%v", alb))
		res = append(res, fmt.Sprintf(alb))
	}
	// if error in rows ie rows.Err()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("allAlbumNames: %v", err)
	}
	// fmt.Println("albums/titles: ", res)
	return res, nil
}

// queryData returns album query in struct eg title- defunct for now
func queryData(item string) ([]Album, error) {
	// An albums slice to hold data from returned rows.
	var albums []Album
	cmd := fmt.Sprint("SELECT * FROM album WHERE title = ?", item)
	rows, err := db.Query(cmd) //see if this works better
	// rows, err := db.Query("SELECT * FROM album WHERE title = ?", item)
	if err != nil {
		return nil, fmt.Errorf("queryData: %v", err)
	}

	//print columns
	col, _ := rows.Columns()
	fmt.Println("Print columns:", col[0])

	defer rows.Close()
	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var alb Album
		if err := rows.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
			return nil, fmt.Errorf("queryData %q: %v", item, err)
		}
		albums = append(albums, alb)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("albumsByArtist %q: %v", item, err)
	}

	return albums, nil
}

// album search by title of al;bum
// albumsByArtist queries for albums that have the specified artist name.
func albumsSearch(name string) ([]Album, error) {
	// An albums slice to hold data from returned rows.
	var albums []Album

	rows, err := db.Query("SELECT * FROM album WHERE title = ?", name)
	if err != nil {
		return nil, fmt.Errorf("albumsSearch %q: %v", name, err)
	}
	defer rows.Close()
	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var alb Album
		if err := rows.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
			return nil, fmt.Errorf("albumsSearch %q: %v", name, err)
		}
		albums = append(albums, alb)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("albumsSearch %q: %v", name, err)
	}
	return albums, nil
}

// search by Price
// albumByID queries for the album with the specified ID.
func albumByPrice(id int64) (Album, error) {
	// An album to hold data from the returned row.
	var alb Album

	row := db.QueryRow("SELECT * FROM album WHERE price BETWEEN ? AND ?+1", id)
	if err := row.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
		if err == sql.ErrNoRows {
			return alb, fmt.Errorf("albumsById %d: no such album", id)
		}
		return alb, fmt.Errorf("albumsById %d: %v", id, err)
	}
	return alb, nil
}
