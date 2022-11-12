package main

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"text/template"

	"github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

var db *sql.DB

// validate templates
var tmpl = template.Must(template.ParseFiles("templates/search.html", "templates/add.html", "templates/delete.html", "templates/dump.html", "templates/test.html"))

// Album struct
type Album struct {
	ID     int64
	Title  string
	Artist string
	Price  float32
}

// Page structure
type Page struct {
	Titles []string
	Body   []Album
	Price  []float32
	Names  []string
}

func main() {
	l := log.WithField("Alpha", "starting up...")
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
		l.Fatal(err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		l.Fatal(pingErr)
	}

	l = log.WithFields(log.Fields{
		"In":      "main()",
		"Action":  "Connect db",
		"DB Name": cfg.DBName,
	})
	l.Info("Connected!\n")

	// TEST in MAIN
	/* 	cmd := "SELECT * FROM album ORDER by artist LIMIT 50;"
	   	q, er := genericQuery(cmd)
	   	if er != nil {
	   		l.Fatalf("bad generic Query %v", er)
	   	}
	   	l.Infof("Results of generic Query: %v", q) */
	/* 	alb := Album{19, "Lessons", "Umfundisi", 2}
	   	editText := "Umfundissssi"
	   	ed, err := updateAlbum(alb, editText)
	   	if err != nil {
	   		l.Fatalf("cant edit bcoz: %v", err)
	   	}
	   	fmt.Println("results: ", ed) */
	//END TEST

	//http call handler:
	http.HandleFunc("/", searchHandler)
	http.HandleFunc("/add", addHandler)
	http.HandleFunc("/delete", deleteHandler)
	http.HandleFunc("/dump", dumpHandler)
	http.HandleFunc("/test", testHandler)
	http.HandleFunc("/edit", editHandler)
	http.HandleFunc("/styles/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "styles/style.css")
	})
	l.Info("Serving http://localhost:8080")
	l.Fatal(http.ListenAndServe(":8080", nil))

}

// albumsByArtist queries for albums that have the specified artist name.
func albumsByArtist(name string) ([]Album, map[string]interface{}, error) {
	// An albums slice to hold data from returned rows.
	var albums []Album
	var albMap map[string]interface{} //use case is mapped key:value db result

	rows, err := db.Query("SELECT * FROM album WHERE artist = ?", name)
	if err != nil {
		return nil, nil, fmt.Errorf("albumsByArtist %q: %v", name, err)
	}
	defer rows.Close()
	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var alb Album
		if err := rows.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
			return nil, nil, fmt.Errorf("albumsByArtist %q: %v", name, err)
		}
		albums = append(albums, alb)
		albMap = map[string]interface{}{
			"ID": alb.ID, "Title": alb.Title, "Artist": alb.Artist, "Price": alb.Price,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("albumsByArtist %q: %v", name, err)
	}
	l := log.WithFields(log.Fields{"in": "albumsByArtist()", "Action": "Fetched albums by artist", "data": albums})
	l.Info()
	return albums, albMap, nil
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

// searchhandler -> search page, results, edit btn
func searchHandler(w http.ResponseWriter, r *http.Request) {

	l := log.WithFields(log.Fields{"IN": "Search Handler"})

	//anyonymous func to handle errors
	check := func(err error, whereAt string) {
		if err != nil {
			l = l.WithField("At", whereAt)
			l.Fatalf("Error at %v: %v", whereAt, err)
		}
	}
	//handle NOT a POST request, render blank search template
	if r.Method != http.MethodPost {
		l = l.WithFields(log.Fields{"template": " blank */search.html"})

		//fetch artists names
		artistsList, err := allArtistNames()
		check(err, "artistsList")

		//fetch dropdown for album titles
		titlesList, err := allAlbumNames()
		check(err, "titlesList")

		//fetch pricelist //TODO
		priceList, err := allAlbumPrices()
		check(err, "priceList")

		// prepare page struct for form dropdowns ->title & artist
		art := Page{
			Titles: titlesList,
			Names:  artistsList,
			Price:  priceList,
		}

		l = l.WithFields(log.Fields{"Action": "form data", "titles": len(art.Titles), "artists": len(art.Names)})
		l.Info()

		// parse blank search template
		tmpl, err = template.ParseFiles("templates/search.html")
		check(err, "parse search template")

		//execute search template with dropdown data
		tmpl.Execute(w, struct {
			Success bool
			Body    Page
		}{false, art})

		l.Info()
		return
	} else {
		// handle form with results
		l = l.WithField("action", "search form results, search.html")

		// get form input values
		var price float32
		var err error
		priceValue := r.FormValue("price")

		// if priceValue input, format priceValue to float32
		if priceValue != "" {
			prc, err := strconv.ParseFloat(priceValue, 32)
			check(err, "priceValue to float32")
			price = float32(prc)
		}

		// save all input form values in album struct
		details := Album{
			Title: r.FormValue("title"), Artist: r.FormValue("artist"), Price: price,
		}

		var albumResult []Album
		var albumResultMap map[string]interface{} // returns mapped key:value pairs for better processing of results

		// conditional data search results in albumResult slice: TODO: use switch statement
		//if we have a price, we must have either artist or title data for search
		if details.Price > 0.00 {
			//if price + artist
			if details.Artist != "" {
				l = l.WithField("where", "price + artist search")
				pArt, pArtMap, err := albumPriceArtist(details.Price, details.Artist)
				check(err, "in price + artist search")
				l = l.WithField("price+artist res", pArt)
				// convert p to album slice
				albumResult = pArt
				albumResultMap = pArtMap
				l.Info("albMapResult: ", albumResultMap)
			} else {
				//else its price only search
				l = l.WithField("where", "at price only search")
				priceOnly, err := albumByPrice(details.Price)
				check(err, "in price only search")
				albumResult = []Album{priceOnly}
				albumResult = []Album{priceOnly}
				l.Info("albMapResult: ", albumResultMap)
			}
		} else if details.Title != "" {
			//TITLE ONLY
			l = l.WithField("where", "title only search")
			albumResult, err = albumsSearch(details.Title)
			check(err, "in title only search")
			l.Info("albMapResult: ", albumResultMap)
		} else if details.Artist != "" {
			//ARTIST ONLY
			l = l.WithField("where", "artist only search")
			albumResult, albumResultMap, err = albumsByArtist(details.Artist)
			check(err, "in album only search")
			l.Info("testing albumMapResult", albumResultMap)
		}

		/* l.Info("(map not be blank ) albumResult: ", albumResult)
		l.Info("albMapResult: ", albMapResult) */
		if len(albumResult) == 0 {
			l.Warn("albumResult shouldnt be blank at this point. expect errors.")
		}
		// put page data in page struct, in slices
		pageInfo := Page{
			Titles: []string{details.Title},
			Names:  []string{details.Artist},
			Price:  []float32{details.Price},
			Body:   albumResult,
		}

		// execute template with search results
		tmpl.Execute(w, struct {
			Success      bool
			Body         Page
			AlbumMap     map[string]interface{}
			ResultTitle  interface{}
			ResultArtist interface{}
			ResultPrice  interface{}
			ResultID     interface{}
		}{true, pageInfo, albumResultMap, albumResultMap["Title"], albumResultMap["Artist"], albumResultMap["Price"], albumResultMap["ID"]})

		l.Info("Parsed & exec search results. ")
	}

	/* // artistvalue as the test
	if artistValue != "" {
		details := Album{
			Title:  titleValue,
			Price:  price,
			Artist: artistValue,
		}
		tmpl.Execute(w, struct {
			Success bool
			Body    Album
		}{
			true, details,
		})
	} else {
		tmpl.Execute(w, struct {
			Success bool
			Message string
		}{
			true, "successful",
		})
	} */

}

// addHandler - handler for add action
func addHandler(w http.ResponseWriter, r *http.Request) {
	var price float32
	var err error
	l := log.WithFields(log.Fields{"In": "Add Handler", "Action": "Parse Template"})

	//parse template
	tmpl, err := template.ParseFiles("templates/add.html")
	if err != nil {
		l.Fatalf("Add album Handler ParseFiles Error: %v", err)
	}

	//fetch vars
	priceStr := r.FormValue("price")
	// if price is passed, round price 2 decimal places
	if priceStr != "" {
		// convert string to float64
		priceValue, err := strconv.ParseFloat(priceStr, 32)
		if err != nil {
			l.WithFields(log.Fields{"value": priceValue, "error": err})
			l.Warnf("In strconv.ParseFloat error %v: ", err)
			l.Fatal(err)
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
			l.Fatalf("Sorry, can't let you add this album because of some error: %v", err)
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
	tmpl, err := template.ParseFiles("templates/delete.html")
	if err != nil {
		l.Fatalf("Delete album Handler ParseFiles Error: %v", err)
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
			l.Fatalf("Sorry, can't let you delete this album because: %v", err)
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
	//need to sort res and all the Db dropdown lists
	sort.Slice(res, func(i, j int) bool {
		return res[i] < res[j]
	})
	/* 	res = type SortBy []Type

	   	func (a SortBy) Len() int           { return len(a) }
	   	func (a SortBy) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
	   	func (a SortBy) Less(i, j int) bool { return a[i] < a[j] } */
	l = l.WithFields(log.Fields{"Result": fmt.Sprintf("%v count", len(res))})
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
	//need to sort res and all the Db dropdown lists
	sort.Slice(res, func(i, j int) bool {
		return res[i] < res[j]
	})

	l = l.WithFields(log.Fields{"Result": fmt.Sprintf("%v count", len(res))})
	l.Info()
	return res, nil
}

// album search by title of album
func albumsSearch(name string) ([]Album, error) {
	// An albums slice to hold data from returned rows.
	var albums []Album

	l := log.WithFields(log.Fields{"func": "albumsSearch()", "title": name})

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
func albumPriceArtist(price float32, artist string) ([]Album, map[string]interface{}, error) {
	// An album to hold data from the returned row.
	var albums []Album
	var albMap map[string]interface{}
	l := log.WithFields(log.Fields{"In": "albumPriceArtist()", "price": price, "artist": artist})
	l.Info("Queue the orchestra...")

	rows, err := db.Query("SELECT * FROM album WHERE price = ? AND artist = ? ORDER BY 1; ", fmt.Sprint(price), artist)
	//if err := row.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, fmt.Errorf("albumPriceArtist %f: no such album", price)
		}
		return nil, nil, fmt.Errorf("albumPriceArtist %f: %v", price, err)
	}
	defer rows.Close()

	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var alb Album
		if err := rows.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
			return nil, nil, fmt.Errorf("albumsByArtist %q: %v", artist, err)
		}
		albums = append(albums, alb)
		albMap = map[string]interface{}{
			"ID": alb.ID, "Title": alb.Title, "Artist": alb.Artist, "Price": alb.Price,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("albumsByArtist %q: %v", artist, err)
	}
	l = l.WithFields(log.Fields{"albumPriceArtist() result": albums})
	l.Info("grammies are in...")
	return albums, albMap, nil
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

	l = l.WithFields(log.Fields{"Price max": res[(len(res) - 1)]})
	l.Info()
	return res, nil
}

// testHandler
func testHandler(w http.ResponseWriter, r *http.Request) {
	l := log.WithFields(log.Fields{"In": "testHandler()"})
	tmpl, err := template.ParseFiles("templates/test.html")
	if err != nil {
		l.Fatalf("test template errors %v", err)
	}
	//fictional prices
	prices := []float32{1.50, 2.50, 3.50, 4.50}
	priceValue := r.FormValue("price")

	//priceValue to []float32
	if priceValue != "" {
		testp := []float32{200.00}
		tmpl.Execute(w, struct {
			Success   bool
			Message   string
			Submitted []float32
		}{true, "Success $", testp})
	} else {
		tmpl.Execute(w, struct {
			Success bool
			Prices  []float32
		}{false, prices})
	}
}

// dumpHandler
func dumpHandler(w http.ResponseWriter, r *http.Request) {
	l := log.WithFields(log.Fields{"in": "dumpHandler()"})

	//fetch data
	data, err := dataDump()
	if err != nil {
		l.Errorf("dumpHandler: %v", err)
	}
	l.Infof("data count %v", len(data))
	details := Page{
		Body: data,
	}
	tmpl, _ := template.ParseFiles("templates/dump.html")
	l.Info()
	tmpl.Execute(w, details)
}

// dataDump
func dataDump() ([]Album, error) {
	//similar to album Name Search
	var albums []Album

	l := log.WithFields(log.Fields{"Result": "dataDump()"})

	rows, err := db.Query("SELECT * FROM album ORDER by artist LIMIT 50;")
	if err != nil {
		return nil, fmt.Errorf("dataDump(): %v", err)
	}
	defer rows.Close()
	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var alb Album
		if err := rows.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
			return nil, fmt.Errorf("dataDump(): %v", err)
		}
		albums = append(albums, alb)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dataDump(): %v", err)
	}
	l = l.WithFields(log.Fields{"Albums": len(albums)})
	l.Info()
	return albums, nil

}

// generic query
func genericQuery(cmd string) ([]Album, error) {
	l := log.WithFields(log.Fields{"In": "genericQuery()"})
	var albums []Album

	rows, err := db.Query(cmd)
	if err != nil {
		return nil, fmt.Errorf("genericQuery(): %v", err)
	}
	defer rows.Close()
	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var alb Album
		if err := rows.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
			return nil, fmt.Errorf("genericQuery(): %v", err)
		}
		albums = append(albums, alb)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("genericQuery(): %v", err)
	}
	l = l.WithFields(log.Fields{"results": len(albums)})
	l.Info()
	return albums, nil
}

// updateAlbum edits specified album stored the database, returns album ID / new entry
func updateAlbum(alb Album, editText string) (int64, error) {
	// cmd := fmt.Sprintf("UPDATE items,month SET items.price=month.price WHERE items.id=month.id;"
	// cmd := fmt.Sprintf("UPDATE items,alb SET items.title=alb.title WHERE items.id=alb.id;")
	//scenario: {19, Lessons, Umfundizi, 2} from results for "umfundizi" search with goal to edit this name
	/* 	alb = Album{19, "Lessons", "Umfundizi", 2}
	   	editText := "Umfundisi" */
	l := log.WithFields(log.Fields{"In": "editAlbum()", "editing": editText, "by": alb.Artist})
	l.Infof("ID:%v, Title:%v, Artist:%v, Price:$%v ", alb.ID, alb.Title, alb.Artist, alb.Price)
	// cmd := fmt.Sprintf("UPDATE recordings.album SET artist=%v WHERE artist=%v AND ID=%v;", editText, alb.Artist, alb.ID)
	// result, err := db.Exec("INSERT INTO album (title, artist, price) VALUES (?, ?, ?)", alb.Title, alb.Artist, alb.Price)
	result, err := db.Exec("UPDATE album SET artist=? WHERE artist=? AND title=?;", editText, alb.Artist, alb.Title)
	if err != nil {
		return 0, fmt.Errorf("editAlbum: %v", err)
	}
	// rows returns the number of rows affected by an update
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("editAlbum: %v", err)
	}
	l.Infof("%v row(s) updated", rows)
	return rows, nil
}

// editHandler
func editHandler(w http.ResponseWriter, r *http.Request) {
	l := log.WithFields(log.Fields{"In": "editHandler()"})
	l.Info()

	// vars from search results/edit input
	editid, _ := strconv.Atoi(r.FormValue("id"))
	id := int64(editid)

	editPrice, _ := strconv.ParseFloat(r.FormValue("price"), 32)
	price := float32(editPrice)

	details := Album{
		id, r.FormValue("title"), r.FormValue("artist"), price,
	}
	//parse template
	tmpl, err := template.ParseFiles("templates/edit.html")
	if err != nil {
		l.Fatalf("edit template errors %v", err)
	}
	l.Info("Template parsed")
	// price check
	priceStr := r.FormValue("price")
	l.Info("priceStr,", priceStr)
	//var price float32
	// var err error

	// if price is passed
	/* 	if priceStr != "" {
	   		// convert string to float64
	   		priceValue, err := strconv.ParseFloat(priceStr, 32)
	   		if err != nil {
	   			l.WithFields(log.Fields{"value": priceValue, "error": err})
	   			l.Fatal(err)
	   		}
	   		// format 2 Decimal places
	   		priceValue = math.Round(100*priceValue) / 100
	   		// format to float32
	   		price = float32(priceValue)
	   		l = l.WithFields(log.Fields{"price": price})
	   		l.Info("Price is right. ")
	   	}
	   	//put artist, title and price values in struct
	   	details := Album{
	   		Title:  r.FormValue("title"),
	   		Artist: r.FormValue("artist"),
	   		Price:  price,
	   	}
	   	l.Infof("details: %v", details)
	*/
	// if details has data, exec template
	if details.Title != "" || details.Artist != "" || details.Price > 0.0 {
		// var editText string = "Umfundisi" //TODO get value passed not hardcoded
		editText := details.Artist

		l.Infof("Show details %v: ", details)
		// run db process here to update table.
		resp, err := updateAlbum(details, editText)
		if err != nil {
			l.Fatalf("%v ", err)
		}
		l.Infof("do we ever get here? details: %v", details)
		//then execute template
		tmpl.Execute(w, struct {
			Success   bool
			Message   string
			Title     string
			Artist    string
			Price     float32
			Submitted Album
		}{true, fmt.Sprintf("Success updating %v", resp), details.Title, details.Artist, details.Price, details})
	} else {
		l.Infof("when no details: %v", details)
		// edit page when nothing to edit
		tmpl.Execute(w, struct {
			Success bool
			Message string
		}{false, "Nothing to edit. Try something else!"})
	}
}
