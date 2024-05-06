package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

// type User struct {
// 	ID      int    `json:"id"`
// 	Name    string `json:"name"`
// 	Email   string `json:"email"`
// 	Contact string `json:"contact"`
// 	Role    string `json:"role"`
// 	LibID   int    `json:"libID"`
// }

type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Contact  string `json:"contact"`
	Role     string `json:"role"`
	LibID    int    `json:"lib_id"`
	Password string `json:"-"`
}

const defaultOwnerEmail = "default_owner@example.com"
const defaultOwnerRole = "owner"

type Library struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type BookInventory struct {
	ISBN            string `json:"isbn"`
	LibID           int    `json:"libID"`
	Title           string `json:"title"`
	Authors         string `json:"authors"`
	Publisher       string `json:"publisher"`
	Version         string `json:"version"`
	TotalCopies     int    `json:"totalCopies"`
	AvailableCopies int    `json:"availableCopies"`
}

type RequestEvent struct {
	ReqID        int       `json:"req_id"`
	BookID       int       `json:"book_id"`
	ReaderID     int       `json:"reader_id"`
	RequestDate  time.Time `json:"request_date"`
	ApprovalDate time.Time `json:"approval_date"`
	ApproverID   int       `json:"approver_id"`
	RequestType  string    `json:"request_type"`
}

type IssueRegistery struct {
	IssueID            int       `json:"issueID"`
	ISBN               string    `json:"isbn"`
	ReaderID           int       `json:"readerID"`
	IssueApproverID    int       `json:"issueApproverID"`
	IssueStatus        string    `json:"issueStatus"`
	IssueDate          time.Time `json:"issueDate"`
	ExpectedReturnDate time.Time `json:"expectedReturnDate"`
	ReturnDate         time.Time `json:"returnDate"`
	ReturnApproverID   int       `json:"returnApproverID"`
}

var db *sql.DB
var err error

func initDatabase() {
	db, err := sql.Open("sqlite3", "library.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check if a default owner user exists
	var defaultUser User
	defaultUser.Email = defaultOwnerEmail
	defaultUser.Role = defaultOwnerRole
	defaultUser.LibID = 1
	err = db.QueryRow("SELECT ID, Name, Contact FROM Users WHERE Email = ? AND Role = ? AND LibID = ?", defaultUser.Email, defaultUser.Role, defaultUser.LibID).Scan(&defaultUser.ID, &defaultUser.Name, &defaultUser.Contact)
	if err != nil {
		fmt.Println("Error querying Users table:", err)
		// If not, create a default owner user
		defaultUser.Name = "Root"
		defaultUser.Contact = "1234567890"
		defaultUser.Password = "password"
		_, err = db.Exec("INSERT INTO Users (Name, Email, ContactNumber, Password, Role, LibID) VALUES (?, ?, ?, ?, ?, ?)", defaultUser.Name, defaultUser.Email, defaultUser.Contact, defaultUser.Password, defaultUser.Role, defaultUser.LibID)
		if err != nil {
			fmt.Println("Error inserting default owner user:", err)
			return
		}
	}
}

func main() {
	db, err = sql.Open("sqlite3", "./library.db")
	if err != nil {
		fmt.Println(err)
	}
	defer db.Close()

	// Ensure the tables exist
	createLibraryTable()
	createUsersTable()
	createBookInventoryTable()
	createRequestEventsTable()
	createIssueRegisteryTable()

	// user routes
	router := gin.Default()
	owner := router.Group("/owner", AuthMiddleware("owner"))
	{
		owner.POST("/library", createLibrary)
		owner.POST("/users", createUser)
	}

	admin := router.Group("/admin", AuthMiddleware("admin"))
	{
		admin.POST("/books", createBook)
		admin.PUT("/books/:isbn", updateBook)
		admin.DELETE("/books/:isbn", deleteBook)
		admin.GET("/requests", listIssues)
		admin.POST("/requests/:reqID", approveIssueRequest)
		admin.GET("/readers/:readerID", getReaderInfo)
	}

	reader := router.Group("/reader", AuthMiddleware("reader"))
	{
		reader.POST("/requests", createRequestEvent)
		reader.GET("/books", listAvailableBooks)
	}

	// router.GET("/users/:id", getUser)
	router.PUT("/users/:id", updateUser)
	router.DELETE("/users/:id", deleteUser)
	router.GET("/users", listUsers)
	// bookInv Routes
	router.POST("/books")
	router.GET("/books/:isbn", getBook)
	router.PUT("/books/:isbn", updateBook)
	router.DELETE("/books/:isbn", deleteBook)
	router.GET("/books", listBooks)
	// Library routes

	router.GET("/library/:id", getLibrary)
	router.PUT("/library/:id", updateLibrary)
	router.DELETE("/library/:id", deleteLibrary)
	router.GET("/library", listLibraries)
	// RequestEvenets Routes
	router.POST("/requestevents", createRequestEvent)
	router.GET("/requestevents/:id", getRequestEvent)
	router.PUT("/requestevents/:id", updateRequestEvent)
	router.DELETE("/requestevents/:id", deleteRequestEvent)
	router.GET("/requestevents", listRequestEvents)

	// IssueReg Routes
	router.POST("/issues", createIssue)
	router.GET("/issues/:issueID", getIssue)
	router.PUT("/issues/:issueID", updateIssue)
	router.DELETE("/issues/:issueID", deleteIssue)
	router.GET("/issues", listIssues)
	router.Run(":8081")
}

func createUsersTable() {
	createUsersTableSQL := `CREATE TABLE IF NOT EXISTS users (
        "ID" INTEGER PRIMARY KEY AUTOINCREMENT,
        "Name" TEXT,
        "Email" TEXT,
        "Contact" TEXT,
        "Role" TEXT,
        "LibID" INTEGER NOT NULL,
        FOREIGN KEY ("LibID") REFERENCES library("ID")
    );`

	_, err := db.Exec(createUsersTableSQL)
	if err != nil {
		fmt.Println(err)
	}
}

func createRequestEventsTable() {
	createRequestEventsTableSQL := `CREATE TABLE IF NOT EXISTS RequestEvents (
        "ReqID" INTEGER PRIMARY KEY AUTOINCREMENT,
        "BookID" INTEGER,
        "ReaderID" INTEGER,
        "RequestDate" DATETIME,
        "ApprovalDate" DATETIME,
        "ApproverID" INTEGER,
        "RequestType" TEXT,
        FOREIGN KEY ("BookID") REFERENCES book_inventory("ISBN"),
        FOREIGN KEY ("ReaderID") REFERENCES users("ID"),
        FOREIGN KEY ("ApproverID") REFERENCES users("ID")
    );`

	_, err := db.Exec(createRequestEventsTableSQL)
	if err != nil {
		fmt.Println(err)
	}
}

func createBookInventoryTable() {
	createBookInventoryTableSQL := `CREATE TABLE IF NOT EXISTS book_inventory (
        "ISBN" TEXT PRIMARY KEY,
        "LibID" INTEGER NOT NULL,
        "Title" TEXT,
        "Authors" TEXT,
        "Publisher" TEXT,
        "Version" TEXT,
        "TotalCopies" INTEGER,
        "AvailableCopies" INTEGER,
        FOREIGN KEY ("LibID") REFERENCES library("ID")
    );`

	_, err := db.Exec(createBookInventoryTableSQL)
	if err != nil {
		fmt.Println(err)
	}
}

func createLibraryTable() {
	createLibraryTableSQL := `CREATE TABLE IF NOT EXISTS library (
        "ID" INTEGER PRIMARY KEY AUTOINCREMENT,
        "Name" TEXT
    );`

	_, err := db.Exec(createLibraryTableSQL)
	if err != nil {
		fmt.Println(err)
	}
}

func createIssueRegisteryTable() {
	createIssueRegisteryTableSQL := `CREATE TABLE IF NOT EXISTS IssueRegistery (
        "IssueID" INTEGER PRIMARY KEY AUTOINCREMENT,
        "ISBN" TEXT,
        "ReaderID" INTEGER,
        "IssueApproverID" INTEGER,
        "IssueStatus" TEXT,
        "IssueDate" DATETIME,
        "ExpectedReturnDate" DATETIME,
        "ReturnDate" DATETIME,
        "ReturnApproverID" INTEGER,
        FOREIGN KEY ("ISBN") REFERENCES book_inventory("ISBN"),
        FOREIGN KEY ("ReaderID") REFERENCES users("ID"),
        FOREIGN KEY ("IssueApproverID") REFERENCES users("ID"),
        FOREIGN KEY ("ReturnApproverID") REFERENCES users("ID")
    );`

	_, err := db.Exec(createIssueRegisteryTableSQL)
	if err != nil {
		fmt.Println(err)
	}
}

func createUser(c *gin.Context) {
	var newUser User

	if err := c.BindJSON(&newUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	statement, _ := db.Prepare("INSERT INTO users (Name, Email, Contact, Role, LibID) VALUES (?,?,?,?,?)")
	result, err := statement.Exec(newUser.Name, newUser.Email, newUser.Contact, newUser.Role, newUser.LibID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	newUser.ID = int(id)
	c.JSON(http.StatusCreated, newUser)
}

func getUser(c *gin.Context) {
	id := c.Param("id")
	var user User

	row := db.QueryRow("SELECT * FROM users WHERE ID =?", id)
	err := row.Scan(&user.ID, &user.Name, &user.Email, &user.Contact, &user.Role, &user.LibID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// AuthMiddleware is a middleware for authentication
func AuthMiddleware(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		email, password, ok := c.Request.BasicAuth()
		if !ok {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		db, err := sql.Open("sqlite3", "library.db")
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		defer db.Close()

		var user User
		err = db.QueryRow("SELECT ID, Name, Email, Contact, Role, LibID, Password FROM Users WHERE Email =?", email).Scan(&user.ID, &user.Name, &user.Email, &user.Contact, &user.Role, &user.LibID, &user.Password)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		if user.Password != password {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		if !strings.EqualFold(user.Role, role) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Set("user", user)
		c.Next()
	}
}

func updateUser(c *gin.Context) {
	id := c.Param("id")
	var user User

	if err := c.BindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec("UPDATE users SET Name =?, Email =?, Contact =?, Role =?, LibID =? WHERE ID =?", user.Name, user.Email, user.Contact, user.Role, user.LibID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user.ID, _ = strconv.Atoi(id)
	c.JSON(http.StatusOK, user)
}

func deleteUser(c *gin.Context) {
	id := c.Param("id")

	_, err := db.Exec("DELETE FROM users WHERE ID =?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

func listUsers(c *gin.Context) {
	var users []User

	rows, err := db.Query("SELECT * FROM users")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Contact, &user.Role, &user.LibID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		users = append(users, user)
	}

	c.JSON(http.StatusOK, users)
}

// BookInventory Creation
func createBook(c *gin.Context) {
	var newBook BookInventory

	if err := c.BindJSON(&newBook); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	statement, _ := db.Prepare(`
	INSERT INTO book_inventory (ISBN, LibID, Title, Authors, Publisher, Version, TotalCopies, AvailableCopies)
	VALUES (?,?,?,?,?,?,?,?)
`)
	_, _ = statement.Exec(newBook.ISBN, newBook.LibID, newBook.Title, newBook.Authors, newBook.Publisher, newBook.Version, newBook.TotalCopies, newBook.AvailableCopies)
	c.JSON(http.StatusCreated, newBook)
}

func getBook(c *gin.Context) {
	isbn := c.Param("isbn")
	var book BookInventory

	row := db.QueryRow("SELECT * FROM book_inventory WHERE ISBN =?", isbn)
	err := row.Scan(&book.ISBN, &book.LibID, &book.Title, &book.Authors, &book.Publisher, &book.Version, &book.TotalCopies, &book.AvailableCopies)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, book)
}

func updateBook(c *gin.Context) {
	isbn := c.Param("isbn")
	var book BookInventory

	if err := c.BindJSON(&book); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec("UPDATE book_inventory SET LibID =?, Title =?, Authors =?, Publisher =?, Version =?, TotalCopies =?, AvailableCopies =? WHERE ISBN =?", book.LibID, book.Title, book.Authors, book.Publisher, book.Version, book.TotalCopies, book.AvailableCopies, isbn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, book)
}

func deleteBook(c *gin.Context) {
	isbn := c.Param("isbn")

	_, err := db.Exec("DELETE FROM book_inventory WHERE ISBN =?", isbn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Book deleted"})
}

func listBooks(c *gin.Context) {
	var books []BookInventory

	rows, err := db.Query("SELECT * FROM book_inventory")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var book BookInventory
		if err := rows.Scan(&book.ISBN, &book.LibID, &book.Title, &book.Authors, &book.Publisher, &book.Version, &book.TotalCopies, &book.AvailableCopies); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		books = append(books, book)
	}

	c.JSON(http.StatusOK, books)
}

// Library
func listLibraries(c *gin.Context) {
	var libraries []Library

	rows, err := db.Query("SELECT * FROM library")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var library Library
		if err := rows.Scan(&library.ID, &library.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		libraries = append(libraries, library)
	}

	c.JSON(http.StatusOK, libraries)
}

func createLibrary(c *gin.Context) {
	var newLibrary Library

	if err := c.BindJSON(&newLibrary); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	statement, _ := db.Prepare("INSERT INTO library (Name) VALUES (?)")
	result, _ := statement.Exec(newLibrary.Name)
	id, _ := result.LastInsertId()

	newLibrary.ID = int(id)
	c.JSON(http.StatusCreated, newLibrary)
}

func getLibrary(c *gin.Context) {
	id := c.Param("id")
	var library Library

	row := db.QueryRow("SELECT * FROM library WHERE ID =?", id)
	err := row.Scan(&library.ID, &library.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Library not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, library)
}

func updateLibrary(c *gin.Context) {
	id := c.Param("id")
	var library Library

	if err := c.BindJSON(&library); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec("UPDATE library SET Name =? WHERE ID =?", library.Name, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	library.ID, _ = strconv.Atoi(id)
	c.JSON(http.StatusOK, library)
}

func deleteLibrary(c *gin.Context) {
	id := c.Param("id")

	_, err := db.Exec("DELETE FROM library WHERE ID =?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Library deleted"})
}

// Request Events

func createRequestEvent(c *gin.Context) {
	var newRequestEvent RequestEvent

	if err := c.BindJSON(&newRequestEvent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	statement, _ := db.Prepare("INSERT INTO RequestEvents (BookID, ReaderID, RequestDate, ApprovalDate, ApproverID, RequestType) VALUES (?,?,?,?,?,?)")
	result, _ := statement.Exec(newRequestEvent.BookID, newRequestEvent.ReaderID, newRequestEvent.RequestDate, newRequestEvent.ApprovalDate, newRequestEvent.ApproverID, newRequestEvent.RequestType)
	id, _ := result.LastInsertId()

	newRequestEvent.ReqID = int(id)
	c.JSON(http.StatusCreated, newRequestEvent)
}

func getRequestEvent(c *gin.Context) {
	id := c.Param("id")
	var requestEvent RequestEvent

	row := db.QueryRow("SELECT * FROM RequestEvents WHERE ReqID =?", id)
	err := row.Scan(&requestEvent.ReqID, &requestEvent.BookID, &requestEvent.ReaderID, &requestEvent.RequestDate, &requestEvent.ApprovalDate, &requestEvent.ApproverID, &requestEvent.RequestType)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "RequestEvent not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, requestEvent)
}

func updateRequestEvent(c *gin.Context) {
	id := c.Param("id")
	var requestEvent RequestEvent

	if err := c.BindJSON(&requestEvent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec("UPDATE RequestEvents SET BookID =?, ReaderID =?, RequestDate =?, ApprovalDate =?, ApproverID =?, RequestType =? WHERE ReqID =?", requestEvent.BookID, requestEvent.ReaderID, requestEvent.RequestDate, requestEvent.ApprovalDate, requestEvent.ApproverID, requestEvent.RequestType, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	requestEvent.ReqID, _ = strconv.Atoi(id)
	c.JSON(http.StatusOK, requestEvent)
}

func deleteRequestEvent(c *gin.Context) {
	id := c.Param("id")

	_, err := db.Exec("DELETE FROM RequestEvents WHERE ReqID =?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "RequestEvent deleted"})
}

func listRequestEvents(c *gin.Context) {
	var requestEvents []RequestEvent

	rows, err := db.Query("SELECT * FROM RequestEvents")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var requestEvent RequestEvent
		if err := rows.Scan(&requestEvent.ReqID, &requestEvent.BookID, &requestEvent.ReaderID, &requestEvent.RequestDate, &requestEvent.ApprovalDate, &requestEvent.ApproverID, &requestEvent.RequestType); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		requestEvents = append(requestEvents, requestEvent)
	}

	c.JSON(http.StatusOK, requestEvents)
}

// Issue registry functions

// Create a new issue registry entry
func createIssue(c *gin.Context) {
	var newIssue IssueRegistery

	if err := c.BindJSON(&newIssue); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	statement, _ := db.Prepare("INSERT INTO IssueRegistery (ISBN, ReaderID, IssueApproverID, IssueStatus, IssueDate, ExpectedReturnDate) VALUES (?,?,?,?,?,?)")
	result, err := statement.Exec(newIssue.ISBN, newIssue.ReaderID, newIssue.IssueApproverID, newIssue.IssueStatus, newIssue.IssueDate, newIssue.ExpectedReturnDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	newIssue.IssueID = int(id)
	c.JSON(http.StatusCreated, newIssue)
}

// Get an issue registry entry by ID
func getIssue(c *gin.Context) {
	id := c.Param("issueID")
	var issue IssueRegistery

	row := db.QueryRow("SELECT * FROM IssueRegistery WHERE IssueID =?", id)
	err := row.Scan(&issue.IssueID, &issue.ISBN, &issue.ReaderID, &issue.IssueApproverID, &issue.IssueStatus, &issue.IssueDate, &issue.ExpectedReturnDate, &issue.ReturnDate, &issue.ReturnApproverID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Issue not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, issue)
}

// Update an issue registry entry
func updateIssue(c *gin.Context) {
	id := c.Param("issueID")
	var updatedIssue IssueRegistery

	if err := c.BindJSON(&updatedIssue); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec("UPDATE IssueRegistery SET ISBN =?, ReaderID =?, IssueApproverID =?, IssueStatus =?, IssueDate =?, ExpectedReturnDate =?, ReturnDate =?, ReturnApproverID =? WHERE IssueID =?", updatedIssue.ISBN, updatedIssue.ReaderID, updatedIssue.IssueApproverID, updatedIssue.IssueStatus, updatedIssue.IssueDate, updatedIssue.ExpectedReturnDate, updatedIssue.ReturnDate, updatedIssue.ReturnApproverID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedIssue)
}

// Delete an issue registry entry
func deleteIssue(c *gin.Context) {
	id := c.Param("issueID")

	_, err := db.Exec("DELETE FROM IssueRegistery WHERE IssueID =?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Issue deleted"})
}

// List all issue registry entries
func listIssues(c *gin.Context) {
	var issues []IssueRegistery

	rows, err := db.Query("SELECT * FROM IssueRegistery")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var issue IssueRegistery
		if err := rows.Scan(&issue.IssueID, &issue.ISBN, &issue.ReaderID, &issue.IssueApproverID, &issue.IssueStatus, &issue.IssueDate, &issue.ExpectedReturnDate, &issue.ReturnDate, &issue.ReturnApproverID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		issues = append(issues, issue)
	}

	c.JSON(http.StatusOK, issues)
}
