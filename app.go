package main

import (
	"database/sql"
	"fmt"
	"math"
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
	Price  []float32
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
		"IN":      "main()",
		"Action":  "Connect db",
		"DB Name": cfg.DBName,
	})
	l.Info("Connected!\n")

	//http calls:
	http.HandleFunc("/", searchHandler)
	http.HandleFunc("/results", resultsHandler)
	http.HandleFunc("/add", addHandler)
	http.HandleFunc("/delete", deleteHandler)
	l.Info("Serving http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

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
	l := log.WithFields(log.Fields{"Action": "Fetched albums by artist", "Result": albums})
	l.Info()
	return albums, nil
}

// albumByID queries for the album with the specified ID.
func albumByID(id int64) (Album, error) {
	// An album to hold data from the returned row.
	var alb Album
	l := log.WithFields(log.Fields{"In": "albumByID()", "id": id})

	row := db.QueryRow("SELECT * FROM album WHERE id = ?", id)
	if err := row.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
		if err == sql.ErrNoRows {
			return alb, fmt.Errorf("albumsById %d: no such album", id)
		}
		l.Infof("error", err)
		return alb, fmt.Errorf("albumsById %d: %v", id, err)
	}
	l.Info("Done")
	return alb, nil
}

// addAlbum adds specified album to the database, returns album ID of new entry
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

// deleteAlbum does delete the album forever
func deleteAlbum(alb Album) (int64, error) {
	l := log.WithFields(log.Fields{"In": "deleteAlbum()", "Album to delete": alb})

	result, err := db.Exec("DELETE FROM album WHERE title = ? AND artist = ?;", alb.Title, alb.Artist)
	// DELETE FROM album WHERE title = 'oner' AND artist = 'cuso';
	if err != nil {
		return 0, fmt.Errorf("deleteAlbum: %v", err)
	}
	// id, err := result.LastInsertId()//see https://dev.mysql.com/doc/refman/8.0/en/delete.html - appropos return values of DELETE
	if err != nil {
		return 0, fmt.Errorf("deleteAlbum: %v", err)
	}
	l.Infof("Deletion result: %v ", result)
	return 1, nil
}

// searchHandler - handler for search
func searchHandler(w http.ResponseWriter, r *http.Request) {
	l := log.WithFields(log.Fields{"IN": "Search Handler"})

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

	priceList, err := allAlbumPrices()
	if err != nil {
		log.Fatal(err)
	}

	// prepare page struct for title and artist dropdowns
	art := Page{
		Titles: titlesList,
		Names:  artistsList,
		Price:  priceList,
	}
	l = l.WithFields(log.Fields{"Action": "have form data", "titles list": art.Titles, "artists list": art.Names})
	l.Info()

	// parse & execute template
	l = l.WithFields(log.Fields{"Action": "Parse Template"})
	l.Info("Parse search template...")
	tmpl, err = template.ParseFiles("search.html")
	if err != nil {
		l.Fatalf("Search Handler ParseFiles Error: %v", err)
	}
	l = l.WithFields(log.Fields{"Action": "Execute Template"})
	l.Info("Execute search template...")
	tmpl.Execute(w, art)
}

// resultHandler - handler for results
func resultsHandler(w http.ResponseWriter, r *http.Request) {
	l := log.WithFields(log.Fields{"IN": "Results Handler"})

	var price float32
	var err error
	priceStr := r.FormValue("price")
	// if price is passed in, round price 2 decimal places
	if priceStr != "" {
		// convert string to float64
		priceValue, err := strconv.ParseFloat(priceStr, 32)
		if err != nil {
			l.WithFields(log.Fields{"value": priceValue, "error": err})
			l.Info("In strconv.ParseFloat...")
			log.Fatal(err)
		}
		// format 2 Decimal places
		priceValue = math.Round(100*priceValue) / 100
		// format to float32
		price = float32(priceValue)
		l = l.WithFields(log.Fields{"price": price})
	}

	//put artist, title and price values in struct
	details := Album{
		Title:  r.FormValue("title"),
		Artist: r.FormValue("artist"),
		Price:  price,
	}

	l = log.WithFields(log.Fields{"details": details})

	//Conditional search - TODO: Switch
	var albumResult []Album
	//if there is price and also title or album input, do multiple search;
	if details.Price > 0.00 {
		//there is either a title or artist included in search
		if details.Title != "" || details.Artist != "" {
			if details.Title != "" {
				// PRICE + TITLE
				result, err := albumPriceTitle(details.Price, details.Title)
				if err != nil {
					l.Warn("Bad query - ", err)
				}
				// cast result to album type for return
				albumResult = []Album{result}
			} else if details.Artist != "" {
				// PRICE + ARTIST
				result, err := albumPriceArtist(details.Price, details.Artist)
				if err != nil {
					log.Warn("Try again with beautiful query.... ", err)
				}
				// cast result to album type for return
				albumResult = []Album{result}
			}
		} else {
			// PRICE ONLY SEARCh
			l.Info("Price search ", details.Price)
			p, err := albumByPrice(details.Price)
			if err != nil {
				log.Fatal(err)
			}
			l.Infof("p: ", p)
			albumResult = []Album{p} //cast result into accepted data type
		}
	} else { //if only title or only artist search, no price
		if details.Title != "" {
			albumResult, err = albumsSearch(details.Title)
			if err != nil {
				log.Fatal(err)
			}
		} else if details.Artist != "" {
			albumResult, err = albumsByArtist(details.Artist)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	l.Info("search criteria met and processed from data strore...")
	// prep page data in page struct we created
	pageInfo := Page{
		Titles: []string{details.Title},
		Names:  []string{details.Artist},
		Price:  []float32{details.Price},
		Body:   albumResult,
	}

	// parse & execute template
	l = l.WithFields(log.Fields{"Action": "Parse & execute template"})
	l.Info()
	tmpl, err = template.ParseFiles("results.html")
	if err != nil {
		log.Fatal(err) //TODO: add more to error log/why failed
	}
	tmpl.Execute(w, pageInfo)
}

// addHandler - handler for add action
func addHandler(w http.ResponseWriter, r *http.Request) {
	var price float32
	var err error
	l := log.WithFields(log.Fields{"In": "Add Handler", "Action": "Parse Template"})

	//parse template
	tmpl, err := template.ParseFiles("add.html")
	if err != nil {
		log.Fatalf("Add album Handler ParseFiles Error: %v", err)
	}

	//fetch vars
	priceStr := r.FormValue("price")
	// if price is passed, round price 2 decimal places
	if priceStr != "" {
		// convert string to float64
		priceValue, err := strconv.ParseFloat(priceStr, 32)
		if err != nil {
			l.WithFields(log.Fields{"value": priceValue, "error": err})
			l.Info("In strconv.ParseFloat...")
			log.Fatal(err)
		}
		// format 2 Decimal places
		priceValue = math.Round(100*priceValue) / 100
		// format to float32
		price = float32(priceValue)
		l = l.WithFields(log.Fields{"price": price})
		l.Info()
	}

	//put artist, title and price values in struct
	details := Album{
		Title:  r.FormValue("title"),
		Artist: r.FormValue("artist"),
		Price:  price,
	}

	//execute conditions 1: if inputs blank(fresh start), render blank template
	if details.Title == "" && details.Artist == "" && details.Price == 0.00 {
		l = l.WithFields(log.Fields{"Action": "Render Add Album Template"})
		l.Info("Render new add album template.")
		tmpl.Execute(w, nil)
	} else {
		//execute condition 2. execute sql and return success msg to client
		id, err := addAlbum(details)
		if err != nil {
			log.Fatalf("Sorry, can't let you add this album because: %v", err)
		}
		l = l.WithFields(log.Fields{"Current Action": "Add Album"})
		l.Infof("Successfully added new album %v by %v $%v (id# %v)", details.Title, details.Artist, details.Price, id)
		tmpl.Execute(w, struct {
			Success bool
			Body    string
		}{true, fmt.Sprintf("%v by %v $%v", details.Title, details.Artist, details.Price)})
	}
}

// deleteHandler - handler for delete action
func deleteHandler(w http.ResponseWriter, r *http.Request) {
	// var price float32
	var err error
	l := log.WithFields(log.Fields{"In": "Delete Handler", "Action": "Parse Template"})

	//parse template
	tmpl, err := template.ParseFiles("delete.html")
	if err != nil {
		log.Fatalf("Delete album Handler ParseFiles Error: %v", err)
	}

	//put artist, title and price values in struct
	details := Album{
		Title:  r.FormValue("title"),
		Artist: r.FormValue("artist"),
		// Price:  price,
	}

	//execute conditions 1: if inputs blank(fresh start), render blank template
	if details.Title == "" && details.Artist == "" {
		l = l.WithFields(log.Fields{"Action": "Render Delete Album Template"})
		l.Info("Render delete album template.")
		tmpl.Execute(w, nil)
	} else {
		//execute condition 2. execute sql and return success msg to client
		id, err := deleteAlbum(details)
		if err != nil {
			log.Fatalf("Sorry, can't let you delete this album because: %v", err)
		}

		var msg string
		l = l.WithFields(log.Fields{"Current Action": "Delete Album"})
		if id == 0 {
			l.Warnf("This album doesnt exist! %v by %v", details.Title, details.Artist)
			msg = fmt.Sprintf("This album doesnt exist! %v by %v", details.Title, details.Artist)
		} else {
			msg = fmt.Sprintf("Successful deletion of album! %v by %v", details.Title, details.Artist)
			l.Infof("Successfully deleted album %v by %v (id# %v)", details.Title, details.Artist, id)
		}
		tmpl.Execute(w, struct {
			Success bool
			Body    string
		}{true, msg})
	}
}

// allArtistNames - helper func to get names of all artists in album table
func allArtistNames() ([]string, error) {
	// res us a slice to hold artist names returned
	var res []string
	l := log.WithFields(log.Fields{"IN": "allArtistNames()"})

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
		res = append(res, fmt.Sprintf(alb))
	}
	// if error in rows ie rows.Err()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("allArtistNames: %v", err)
	}
	l = l.WithFields(log.Fields{"Result": res})
	l.Info()
	return res, nil
}

// albumNames
func allAlbumNames() ([]string, error) {
	var res []string
	l := log.WithFields(log.Fields{"IN": "allAlbumNames()"})
	// db query - distinct, no overlap
	cmd := "SELECT title from album;"
	rows, err := db.Query(cmd)
	if err != nil {
		return nil, fmt.Errorf("allAlbumNames: %v", err)
	}

	defer rows.Close()
	//loop through rows, put names in slice of strings we created
	var alb string // temp string to store distinct artist names
	//if data in rows exists
	for rows.Next() {
		if err := rows.Scan(&alb); err != nil {
			return nil, fmt.Errorf("In allAlbumNames: %v", err)
		}
		res = append(res, fmt.Sprintf(alb))
	}
	// if error in rows ie rows.Err()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("allAlbumNames: %v", err)
	}
	l = l.WithFields(log.Fields{"Result": res})
	l.Info()
	return res, nil
}

// album search by title of album
func albumsSearch(name string) ([]Album, error) {
	// An albums slice to hold data from returned rows.
	var albums []Album

	l := log.WithFields(log.Fields{"Result": "albumsSearch()", "title": name})

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
	l = l.WithFields(log.Fields{"Result": albums})
	l.Info()
	return albums, nil
}

// albumByPrice queries for the album by price.
func albumByPrice(price float32) (Album, error) {
	// An album to hold data from the returned row.
	var alb Album
	l := log.WithFields(log.Fields{"In": "albumByPrice()", "price": price})

	//db query - Note: converting price to string for query
	row := db.QueryRow("SELECT * FROM album WHERE price = ?; ", fmt.Sprint(price))
	if err := row.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
		if err == sql.ErrNoRows {
			return alb, fmt.Errorf("albumsByPrice %f: no such album", price)
		}
		return alb, fmt.Errorf("albumsByPrice %f: %v", price, err)
	}
	l = l.WithFields(log.Fields{"Album result": alb})
	l.Info()
	return alb, nil
}

// albumByPriceTitle queries for album by price+title. --not in use i dont think
func albumPriceTitle(price float32, title string) (Album, error) {
	// An album to hold data from the returned row.
	var alb Album
	l := log.WithFields(log.Fields{"In": "albumByPrice()", "price": price})

	//db query - Note: converting price to string for query
	row := db.QueryRow("SELECT * FROM album WHERE price = ? AND title = ?; ", fmt.Sprint(price), title)
	if err := row.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
		if err == sql.ErrNoRows {
			return alb, fmt.Errorf("albumsByPrice %f: no such album", price)
		}
		return alb, fmt.Errorf("albumsByPrice %f: %v", price, err)
	}
	l = l.WithFields(log.Fields{"Album result": alb})
	l.Info()
	return alb, nil
}

// albumByPriceArtist queries for album by price+artist.
func albumPriceArtist(price float32, artist string) (Album, error) {
	// An album to hold data from the returned row.
	var alb Album
	l := log.WithFields(log.Fields{"In": "albumPriceArtist()", "price": price, "artist": artist})
	l.Info(" queue the orchestra...")

	row := db.QueryRow("SELECT * FROM album WHERE price = ? AND artist = ? ORDER BY 1; ", fmt.Sprint(price), artist)
	if err := row.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
		if err == sql.ErrNoRows {
			return alb, fmt.Errorf("albumPriceArtist %f: no such album", price)
		}
		return alb, fmt.Errorf("albumPriceArtist %f: %v", price, err)
	}
	l = l.WithFields(log.Fields{"albumPriceArtist() result": alb})
	l.Info("grammies are in...")
	return alb, nil
}

// allAlbumPrices - returns album price List
func allAlbumPrices() ([]float32, error) {
	var res []float32
	l := log.WithFields(log.Fields{"IN": "allAlbumPrices()"})

	// db query
	cmd := "SELECT price from album ORDER BY 1;" //ASC
	rows, err := db.Query(cmd)
	if err != nil {
		return nil, fmt.Errorf("allAlbumPrices: %v", err)
	}

	defer rows.Close()
	var alb float32 // temp string to store prices
	//if data in rows exists
	for rows.Next() {
		if err := rows.Scan(&alb); err != nil {
			return nil, fmt.Errorf("In allAlbumPrices: %v", err)
		}
		res = append(res, alb)
	}
	// if error in rows ie rows.Err()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("allAlbumPrices: %v", err)
	}
	l = l.WithFields(log.Fields{"Album Prices": res})
	l.Info()
	return res, nil
}
