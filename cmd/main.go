package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Defines a "model" that we can use to communicate with the
// frontend or the database
// More on these "tags" like `bson:"_id,omitempty"`: https://go.dev/wiki/Well-known-struct-tags
type BookStore struct {
	MongoID     primitive.ObjectID `bson:"_id,omitempty"`
	ID          string
	BookName    string
	BookAuthor  string
	BookEdition string
	BookPages   string
	BookYear    string
}

// Wraps the "Template" struct to associate a necessary method
// to determine the rendering procedure
type Template struct {
	tmpl *template.Template
}

// Preload the available templates for the view folder.
// This builds a local "database" of all available "blocks"
// to render upon request, i.e., replace the respective
// variable or expression.
// For more on templating, visit https://jinja.palletsprojects.com/en/3.0.x/templates/
// to get to know more about templating
// You can also read Golang's documentation on their templating
// https://pkg.go.dev/text/template
func loadTemplates() *Template {
	return &Template{
		tmpl: template.Must(template.ParseGlob("views/*.html")),
	}
}

// Method definition of the required "Render" to be passed for the Rendering
// engine.
// Contraire to method declaration, such syntax defines methods for a given
// struct. "Interfaces" and "structs" can have methods associated with it.
// The difference lies that interfaces declare methods whether struct only
// implement them, i.e., only define them. Such differentiation is important
// for a compiler to ensure types provide implementations of such methods.
func (t *Template) Render(w io.Writer, name string, data interface{}, ctx echo.Context) error {
	return t.tmpl.ExecuteTemplate(w, name, data)
}

// Here we make sure the connection to the database is correct and initial
// configurations exists. Otherwise, we create the proper database and collection
// we will store the data.
// To ensure correct management of the collection, we create a return a
// reference to the collection to always be used. Make sure if you create other
// files, that you pass the proper value to ensure communication with the
// database
// More on what bson means: https://www.mongodb.com/docs/drivers/go/current/fundamentals/bson/
func prepareDatabase(client *mongo.Client, dbName string, collecName string) (*mongo.Collection, error) {
	db := client.Database(dbName)

	names, err := db.ListCollectionNames(context.TODO(), bson.D{{}})
	if err != nil {
		return nil, err
	}
	if !slices.Contains(names, collecName) {
		cmd := bson.D{{"create", collecName}}
		var result bson.M
		if err = db.RunCommand(context.TODO(), cmd).Decode(&result); err != nil {
			log.Fatal(err)
			return nil, err
		}
	}

	coll := db.Collection(collecName)
	return coll, nil
}

// Here we prepare some fictional data and we insert it into the database
// the first time we connect to it. Otherwise, we check if it already exists.
func prepareData(client *mongo.Client, coll *mongo.Collection) {
	startData := []BookStore{
		{
			ID:          "example1",
			BookName:    "The Vortex",
			BookAuthor:  "José Eustasio Rivera",
			BookEdition: "958-30-0804-4",
			BookPages:   "292",
			BookYear:    "1924",
		},
		{
			ID:          "example2",
			BookName:    "Frankenstein",
			BookAuthor:  "Mary Shelley",
			BookEdition: "978-3-649-64609-9",
			BookPages:   "280",
			BookYear:    "1818",
		},
		{
			ID:          "example3",
			BookName:    "The Black Cat",
			BookAuthor:  "Edgar Allan Poe",
			BookEdition: "978-3-99168-238-7",
			BookPages:   "280",
			BookYear:    "1843",
		},
	}

	// This syntax helps us iterate over arrays. It behaves similar to Python
	// However, range always returns a tuple: (idx, elem). You can ignore the idx
	// by using _.
	// In the topic of function returns: sadly, there is no standard on return types from function. Most functions
	// return a tuple with (res, err), but this is not granted. Some functions
	// might return a ret value that includes res and the err, others might have
	// an out parameter.
	for _, book := range startData {
		cursor, err := coll.Find(context.TODO(), book)
		var results []BookStore
		if err = cursor.All(context.TODO(), &results); err != nil {
			panic(err)
		}
		if len(results) > 1 {
			log.Fatal("more records were found")
		} else if len(results) == 0 {
			result, err := coll.InsertOne(context.TODO(), book)
			if err != nil {
				panic(err)
			} else {
				fmt.Printf("%+v\n", result)
			}

		} else {
			for _, res := range results {
				cursor.Decode(&res)
				fmt.Printf("%+v\n", res)
			}
		}
	}
}

// Generic method to perform "SELECT * FROM BOOKS" (if this was SQL, which
// it is not :D ), and then we convert it into an array of map. In Golang, you
// define a map by writing map[<key type>]<value type>{<key>:<value>}.
// interface{} is a special type in Golang, basically a wildcard...
func findAllBooks(coll *mongo.Collection) []map[string]interface{} {
	cursor, err := coll.Find(context.TODO(), bson.D{{}})
	var results []BookStore
	if err = cursor.All(context.TODO(), &results); err != nil {
		panic(err)
	}

	var ret []map[string]interface{}
	for _, res := range results {
		ret = append(ret, map[string]interface{}{
			"ID":          res.ID,
			"BookName":    res.BookName,
			"BookAuthor":  res.BookAuthor,
			"BookEdition": res.BookEdition,
			"BookPages":   res.BookPages,
			"BookYear":    res.BookYear,
		})
	}

	return ret
}

func main() {
	// Connect to the database. Such defer keywords are used once the local
	// context returns; for this case, the local context is the main function
	// By user defer function, we make sure we don't leave connections
	// dangling despite the program crashing. Isn't this nice? :D
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// TODO: make sure to pass the proper username, password, and port
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://mongodb:testmongo@localhost:27017/?authSource=admin"))

	// This is another way to specify the call of a function. You can define inline
	// functions (or anonymous funcstions, similar to the behavior in Python)
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	// You can use such name for the database and collection, or come up with
	// one by yourself!
	coll, err := prepareDatabase(client, "exercise-1", "information")

	if err != nil {
		log.Fatal(err)
	}

	prepareData(client, coll)

	// Here we prepare the server
	e := echo.New()

	// Define our custom renderer
	e.Renderer = loadTemplates()

	// Log the requests. Please have a look at echo's documentation on more
	// middleware
	e.Use(middleware.Logger())

	e.Static("/css", "css")

	// Endpoint definition. Here, we divided into two groups: top-level routes
	// starting with /, which usually serve webpages. For our RESTful endpoints,
	// we prefix the route with /api to indicate more information or resources
	// are available under such route.
	e.GET("/", func(c echo.Context) error {
		return c.Render(200, "index", nil)
	})

	e.GET("/books", func(c echo.Context) error {
		books := findAllBooks(coll)
		return c.Render(200, "book-table", books)
	})

	e.GET("/authors", func(c echo.Context) error {
		books := findAllBooks(coll)
		authorSet := make(map[string]struct{})
		for _, book := range books {
			if author, ok := book["BookAuthor"].(string); ok {
				authorSet[author] = struct{}{}
			}
		}

		var authors []string
		for author := range authorSet {
			authors = append(authors, author)
		}

		return c.Render(200, "authors-table", authors)
	})

	e.GET("/years", func(c echo.Context) error {
		books := findAllBooks(coll)
		yearSet := make(map[string]struct{})
		for _, book := range books {
			if year, ok := book["BookYear"].(string); ok {
				yearSet[year] = struct{}{}
			}
		}

		var years []string
		for year := range yearSet {
			years = append(years, year)
		}

		return c.Render(200, "years-table", years)
	})

	e.GET("/search", func(c echo.Context) error {
		return c.Render(200, "search-bar", nil)
	})

	e.GET("/create", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	// You will have to expand on the allowed methods for the path
	// `/api/route`, following the common standard.
	// A very good documentation is found here:
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Methods
	// It specifies the expected returned codes for each type of request
	// method.
	e.GET("/api/books", func(c echo.Context) error {
		books := findAllBooks(coll)
		var response []map[string]interface{}
		for _, book := range books {
			formatted := map[string]interface{}{
				"id":      book["ID"],
				"title":   book["BookName"],
				"author":  book["BookAuthor"],
				"pages":   book["BookPages"],
				"edition": book["BookEdition"],
				"year":    book["BookYear"],
			}
			response = append(response, formatted)
		}
		return c.JSON(http.StatusOK, response)
	})

	e.POST("/api/books", func(c echo.Context) error {
		var input map[string]interface{}
		if err := c.Bind(&input); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		// Vérifier les champs obligatoires
		id, ok1 := input["id"].(string)
		title, ok2 := input["title"].(string)
		author, ok3 := input["author"].(string)

		if !ok1 || !ok2 || !ok3 || id == "" || title == "" || author == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "id, title and author are required",
			})
		}

		// Construire un document à insérer
		book := map[string]interface{}{
			"ID":          id,
			"BookName":    title,
			"BookAuthor":  author,
			"BookPages":   input["pages"],
			"BookEdition": input["edition"],
			"BookYear":    input["year"],
		}

		// Vérifier si un livre identique existe déjà
		filter := bson.M{
			"ID":          id,
			"BookName":    title,
			"BookAuthor":  author,
			"BookEdition": input["edition"],
			"BookPages":   input["pages"],
			"BookYear":    input["year"],
		}

		count, err := coll.CountDocuments(context.TODO(), filter)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "database error",
			})
		}

		if count > 0 {
			return c.JSON(http.StatusConflict, map[string]string{
				"error": "duplicate book entry",
			})
		}

		// Insérer dans MongoDB
		_, err = coll.InsertOne(context.TODO(), book)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "could not insert book",
			})
		}

		// Retourner 201 Created
		return c.JSON(http.StatusCreated, map[string]string{
			"message": "book created",
		})
	})

	e.GET("/api/books/:id", func(c echo.Context) error {
		bookID := c.Param("id")

		filter := bson.M{"ID": bookID}

		var book BookStore
		err := coll.FindOne(context.TODO(), filter).Decode(&book)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return c.JSON(http.StatusNotFound, map[string]string{
					"error": fmt.Sprintf("Book with ID: %s not found. Is it stored?", bookID),
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "database error",
			})
		}

		// Construire la réponse JSON
		response := map[string]interface{}{
			"id":      book.ID,
			"title":   book.BookName,
			"author":  book.BookAuthor,
			"edition": book.BookEdition,
			"pages":   book.BookPages,
			"year":    book.BookYear,
		}

		return c.JSON(http.StatusOK, response)
	})

	e.PUT("/api/books/:id", func(c echo.Context) error {
		// Récupérer l'ID depuis l'URL
		bookID := c.Param("id")

		// Lire le corps de la requête JSON
		var input map[string]interface{}
		if err := c.Bind(&input); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		// Créer le filtre pour trouver le bon livre
		filter := bson.M{"ID": bookID}

		// Créer le document à mettre à jour (uniquement les champs envoyés)
		update := bson.M{
			"$set": bson.M{},
		}

		// Champs possibles à mettre à jour
		fields := []string{"title", "author", "edition", "pages", "year"}

		for _, field := range fields {
			if val, ok := input[field]; ok {
				// Adapter les noms aux champs en base
				var mongoField string
				switch field {
				case "title":
					mongoField = "BookName"
				case "author":
					mongoField = "BookAuthor"
				case "edition":
					mongoField = "BookEdition"
				case "pages":
					mongoField = "BookPages"
				case "year":
					mongoField = "BookYear"
				}
				update["$set"].(bson.M)[mongoField] = val
			}
		}

		// Effectuer la mise à jour
		result, err := coll.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update book"})
		}

		// Aucun livre trouvé avec cet ID ?
		if result.MatchedCount == 0 {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "book not found"})
		}

		// Succès
		return c.JSON(http.StatusOK, map[string]string{"message": "book updated"})
	})

	e.DELETE("/api/books/:id", func(c echo.Context) error {
		// Récupérer l'ID logique depuis l'URL
		bookID := c.Param("id")

		// Créer un filtre pour chercher le bon livre
		filter := bson.M{"ID": bookID}

		// Supprimer le document
		result, err := coll.DeleteOne(context.TODO(), filter)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "could not delete book",
			})
		}

		// Si aucun document supprimé, c’est que le livre n’existait pas
		if result.DeletedCount == 0 {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "book not found",
			})
		}

		// Suppression réussie
		return c.JSON(http.StatusOK, map[string]string{
			"message": "book deleted",
		})
	})

	// We start the server and bind it to port 3030. For future references, this
	// is the application's port and not the external one. For this first exercise,
	// they could be the same if you use a Cloud Provider. If you use ngrok or similar,
	// they might differ.
	// In the submission website for this exercise, you will have to provide the internet-reachable
	// endpoint: http://<host>:<external-port>
	e.Logger.Fatal(e.Start(":3030"))
}
