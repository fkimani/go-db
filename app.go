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
	fmt.Println("a: ", a)
	a, _ := albumByPrice(17.99)
	fmt.Println("Price SEarch: ", a)
	*/
	// ALBUM PRICES
	/* pricelist, _ := allAlbumPrices()
	fmt.Println("pricelist: ", pricelist) */

	/* //test album by price
	a, err := albumByPrice(17.99)
	if err != nil {
		log.Warnf("Check out this error %v", err)
	}
	fmt.Println("in main - $17.99 album found is: ", a) */
	//so we need to convert 17.990000 to 2 decimal places:
	// option 1: wait...sprintf format it in db query string.
	//option 2. convert it to string, to float 64, round , then back to float32.

	//http calls:
	http.HandleFunc("/", searchHandler)
	http.HandleFunc("/results", resultsHandler)
	l.Info("Serving http://localhost:8080")
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
	// priceList := float32(price)

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
		log.Fatalf("Search Handler ParseFiles Error: %v", err) //TODO: add more to error log/why failed
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

	l = log.WithFields(log.Fields{"details": details})
	l.Info("Details struct ...")

	//Conditional search - TODO: Switch
	var albumResult []Album
	//if there is price and also title or album input, do multiple search;
	//multi search
	if details.Price > 0.00 {
		//there is either a title or artist included in search
		if details.Title != "" || details.Artist != "" {
			if details.Title != "" {
				// PRICE + TITLE
				result, err := albumPriceTitle(details.Price, details.Title)
				if err != nil {
					log.Warn("Bad query - ", err)
					// Album{1 Blue Train John Coltrane 56.99}
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
	//print columns
	/* col, _ := rows.Columns()
	fmt.Println("Print columns:", col[0]) */

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
	//print columns
	/* 	col, _ := rows.Columns()
	   	fmt.Println("Print columns:", col[0]) */

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

// albums search by Price
// albumByID queries for the album with the specified ID.
func albumByPrice(price float32) (Album, error) {
	// An album to hold data from the returned row.
	var alb Album
	l := log.WithFields(log.Fields{"In": "albumByPrice()", "price": price})
	// p := fmt.Sprintf("%.2f", price)
	// fmt.Println("the p", p)
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

// multisearch price + title or price + artist
// albums search by Price
// albumByPriceTitle queries for album by price+title.
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
	// p := fmt.Sprintf("%.2f", price)
	// fmt.Println("the p", p)
	//db query - Note: converting price to string for query
	// test: SELECT * FROM album WHERE price = 17.99 AND artist = GErry Mulligan ORDER BY 1;
	// SELECT * FROM album WHERE price > 49.99 AND artist = 'JoHN ColtraNe' ORDER BY price;
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
		// res = append(res, fmt.Sprintf(alb))
	}
	// if error in rows ie rows.Err()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("allAlbumPrices: %v", err)
	}
	l = l.WithFields(log.Fields{"Album Prices": res})
	l.Info()
	return res, nil
}

// queryData returns album query in struct eg title- defunct for now
/* func queryData(item string) ([]Album, error) {
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
*/
