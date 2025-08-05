package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"slices"
	"strings"

	"google.golang.org/grpc"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"hcorreia/live-blog/blog-service/database"
	"hcorreia/live-blog/common"
	pb "hcorreia/live-blog/common/blog"
)

type server struct {
	db *sql.DB
	pb.UnimplementedBlogServiceServer
}

func NewServer(db *sql.DB) *server {
	return &server{
		db: db,
	}
}

func (s *server) ListPosts(_ context.Context, in *pb.Params) (*pb.PostListResponse, error) {
	log.Printf("Received: %v", in)
	// return &pb.PostResponse{
	// 	ID:        1,
	// 	Title:     "Post #1",
	// 	Image:     "/image-1.jpg",
	// 	Content:   "Content...",
	// 	CreatedAt: "",
	// 	UpdatedAt: "",
	// }, nil

	posts, err := getPosts(s.db)
	if err != nil {
		return nil, err
	}

	result := make([]*pb.PostResponse, 0)

	for _, p := range posts {
		result = append(result, &pb.PostResponse{
			ID:        p.ID,
			Title:     p.Title,
			Image:     p.Image.ValueOrZero(),
			Content:   p.Content,
			CreatedAt: p.CreatedAt.String(),
			UpdatedAt: p.UpdatedAt.String(),
		})
	}

	return &pb.PostListResponse{
		Posts: result,
	}, nil
}

func connectDB() *sql.DB {
	// db, err := sql.Open("mysql", fmt.Sprintf("%s?parseTime=true", os.Getenv("DB_STRING")))
	db, err := sql.Open("mysql", fmt.Sprintf("%s?parseTime=true", common.Env.BlogServiceDbString))
	if err != nil {
		panic(err)
	}

	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(20)

	return db
}

func mainMigration(db *sql.DB, direction string) error {

	fmt.Println("Running migrations",
		strings.ToUpper(direction),
		"...")

	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://./database/migrations",
		"mysql", driver)
	if err != nil {
		return err
	}

	if direction == "status" {
		ver, _, err := m.Version()

		if err == nil {
			fmt.Println("Version: ", ver)
		} else {
			fmt.Println("Version: ", err.Error())
		}

	} else if direction == "up" {
		err = m.Up()
	} else if direction == "down" {
		// err = m.Down()
		err = m.Steps(-1)
	} else {
		panic("Wrong direction!")
	}

	if err != nil {
		fmt.Println("Done:", err.Error())
		// return err
	} else {
		fmt.Println("Done")
	}

	return nil
}

func getPosts(db *sql.DB) ([]database.Post, error) {
	ctx := context.Background()

	queries := database.New(db)

	// list all posts
	posts, err := queries.ListPosts(ctx)
	if err != nil {
		return nil, err
	}
	// log.Println(posts)

	return posts, nil
}

type DataResultMeta struct {
	Timestamp string `json:"timestamp"`
	Hostname  string `json:"hostname"`
}

type DataResult[T any] struct {
	Data T              `json:"data"`
	Meta DataResultMeta `json:"meta"`
}

func panicHelpText() {
	panic(
		"Wrong command line args.\n" +
			"E.g.:\n" +
			"  - backend migrate up\n" +
			"  - backend migrate down\n")
}

func main() {
	if len(os.Args) > 1 {
		if len(os.Args) == 3 &&
			os.Args[1] == "migrate" &&
			slices.Contains([]string{"up", "down", "status"}, os.Args[2]) {

			err := mainMigration(connectDB(), os.Args[2])
			if err != nil {
				panic(err)
			}
			return
		} else {
			panicHelpText()
		}
	}

	db := connectDB()

	lis, err := net.Listen("tcp", common.Env.BlogServiceAddr)
	if err != nil {
		log.Fatalln("Cannot listen: ", err)
	}
	defer lis.Close()

	s := grpc.NewServer()

	pb.RegisterBlogServiceServer(s, NewServer(db))

	log.Printf("server listening at %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		log.Fatalln(err.Error())
	}

}
