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
	"github.com/guregu/null/v6"

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

func (s *server) ListPosts(_ context.Context, in *pb.ListParams) (*pb.PostListResponse, error) {
	log.Printf("ListPosts Received: %v", in)
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

func (s *server) GetPost(_ context.Context, in *pb.IdParam) (*pb.PostResponse, error) {
	log.Printf("GetPost Received: %v", in)
	// return &pb.PostResponse{
	// 	ID:        1,
	// 	Title:     "Post #1",
	// 	Image:     "/image-1.jpg",
	// 	Content:   "Content...",
	// 	CreatedAt: "",
	// 	UpdatedAt: "",
	// }, nil

	post, err := getPost(s.db, in.Id)
	if err != nil {
		return nil, err
	}

	return &pb.PostResponse{
		ID:        post.ID,
		Title:     post.Title,
		Image:     post.Image.ValueOrZero(),
		Content:   post.Content,
		CreatedAt: post.CreatedAt.String(),
		UpdatedAt: post.UpdatedAt.String(),
	}, nil
}

func (s *server) CreatePost(_ context.Context, in *pb.PostCreateRequest) (*pb.PostResponse, error) {
	log.Printf("CreatePost Received: %v", in)
	// return &pb.PostResponse{
	// 	ID:        1,
	// 	Title:     "Post #1",
	// 	Image:     "/image-1.jpg",
	// 	Content:   "Content...",
	// 	CreatedAt: "",
	// 	UpdatedAt: "",
	// }, nil

	post, err := newPost(s.db, database.CreatePostParams{
		Title:   in.Title,
		Image:   null.NewString(in.Image, true),
		Content: in.Content,
	})
	if err != nil {
		return nil, err
	}

	return &pb.PostResponse{
		ID:        post.ID,
		Title:     post.Title,
		Image:     post.Image.ValueOrZero(),
		Content:   post.Content,
		CreatedAt: post.CreatedAt.String(),
		UpdatedAt: post.UpdatedAt.String(),
	}, nil
}

func (s *server) UpdatePost(_ context.Context, in *pb.PostUpdateRequest) (*pb.PostResponse, error) {
	log.Printf("UpdatePost Received: %v", in)
	// return &pb.PostResponse{
	// 	ID:        1,
	// 	Title:     "Post #1",
	// 	Image:     "/image-1.jpg",
	// 	Content:   "Content...",
	// 	CreatedAt: "",
	// 	UpdatedAt: "",
	// }, nil

	post, err := updatePost(s.db, database.UpdatePostParams{
		ID:      in.ID,
		Title:   in.Title,
		Image:   null.NewString(in.Image, true),
		Content: in.Content,
	})
	if err != nil {
		return nil, err
	}

	return &pb.PostResponse{
		ID:        post.ID,
		Title:     post.Title,
		Image:     post.Image.ValueOrZero(),
		Content:   post.Content,
		CreatedAt: post.CreatedAt.String(),
		UpdatedAt: post.UpdatedAt.String(),
	}, nil
}

func (s *server) DeletePost(_ context.Context, in *pb.IdParam) (*pb.Empty, error) {
	log.Printf("DeletePost Received: %v", in)
	// return &pb.PostResponse{
	// 	ID:        1,
	// 	Title:     "Post #1",
	// 	Image:     "/image-1.jpg",
	// 	Content:   "Content...",
	// 	CreatedAt: "",
	// 	UpdatedAt: "",
	// }, nil

	if err := deletePostByID(s.db, in.Id); err != nil {
		return nil, err
	}

	return &pb.Empty{}, nil
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

func getPost(db *sql.DB, id int32) (database.Post, error) {
	ctx := context.Background()

	queries := database.New(db)

	// get one post
	post, err := queries.GetPost(ctx, id)
	if err != nil {
		return database.Post{}, err
	}
	// log.Println(post)

	return post, nil
}

func newPost(db *sql.DB, data database.CreatePostParams) (database.Post, error) {
	ctx := context.Background()

	queries := database.New(db)

	// create post
	result, err := queries.CreatePost(ctx, data)
	if err != nil {
		return database.Post{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return database.Post{}, err
	}

	// fmt.Println("New post ID: ", id, int32(id))

	return getPost(db, int32(id))
}

func updatePost(db *sql.DB, data database.UpdatePostParams) (database.Post, error) {
	ctx := context.Background()

	queries := database.New(db)

	// update post
	if err := queries.UpdatePost(ctx, data); err != nil {
		return database.Post{}, err
	}

	return getPost(db, data.ID)
}

func deletePost(db *sql.DB, data database.Post) error {
	ctx := context.Background()

	queries := database.New(db)

	// delete post
	return queries.DeletePost(ctx, data.ID)
}

func deletePostByID(db *sql.DB, id int32) error {
	ctx := context.Background()

	queries := database.New(db)

	// delete post
	return queries.DeletePost(ctx, id)
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
