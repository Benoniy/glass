package main

import (
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

var filetypes = []string{".md"}


type TestPage struct {
	FileContent template.HTML
	SideBar FileNode
}

// FileNode represents a file or folder in the tree structure
type FileNode struct {
	Name     string
	FullPath string
	IsDir    bool
	Children []*FileNode
}

// buildTree creates a hierarchical file tree structure from a list of paths
func buildTree(paths []string) *FileNode {
	root := &FileNode{Name: "root", IsDir: true}

	for _, path := range paths {
		parts := strings.Split(path, "\\")
		current := root

		for i, part := range parts {
			found := false
			for _, child := range current.Children {
				if child.Name == part {
					current = child
					found = true
					break
				}
			}

			if !found {
				newNode := &FileNode{
					Name:     part,
					FullPath: strings.Join(parts[:i+1], "/"),
					IsDir:    i < len(parts)-1, // If not last, it's a directory
				}
				current.Children = append(current.Children, newNode)
				current = newNode
			}
		}
	}
	sortTree(root)
	return root
}

// sortTree recursively sorts the tree (folders first, then alphabetically)
func sortTree(node *FileNode) {
	sort.Slice(node.Children, func(i, j int) bool {
		if node.Children[i].IsDir != node.Children[j].IsDir {
			return node.Children[i].IsDir // Directories first
		}
		return node.Children[i].Name < node.Children[j].Name // Alphabetical
	})

	for _, child := range node.Children {
		sortTree(child) // Recursively sort subdirectories
	}
}

func mdToHTML(md []byte) []byte {
	// create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock | parser.HardLineBreak
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags
	opts := html.RendererOptions{Flags: htmlFlags, RenderNodeHook: myRenderHook}
	renderer := html.NewRenderer(opts)

	return markdown.Render(doc, renderer)
}

func fileExtInString(ext string) bool {
	for _, e := range filetypes {
		if ext == e {
			return true
		}
	}
	return false
}

func loadMd(title string) []byte {
	filename := title
	file_ext := title[len(title) - 3:]

	if !fileExtInString(file_ext) {
		filename += file_ext
	}
	
	prefix := ""
	if len(title) > len("content/") && title[:len("content/")] != "content/" {
		prefix = "content/"
	}
    
	body, err := os.ReadFile(prefix + filename)
    results := strings.Split(filename, "/")
	result := results[len(results)-1]

	newBody := "# " + result[:len(result)-3] + "  \n" + string(body)
    if err != nil {
        return nil
    }
    return []byte(newBody)
}

// // an actual rendering of Paragraph is more complicated
// func renderLink(w io.Writer, l *ast.Link, entering bool) {
// 	if entering {
// 		print()
// 		io.WriteString(w, "<a href=\"/" + string(l.Destination) + "\">")
// 	} else {
// 		io.WriteString(w, "</a>")
// 	}
// }

func myRenderHook(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	// if link, ok := node.(*ast.Link); ok {
	// 	renderLink(w, link, entering)
	// 	return ast.GoToNext, true
	// }
	return ast.GoToNext, false
}


func viewHandler(w http.ResponseWriter, r *http.Request) {

	var filepaths []string

	filepath.WalkDir("content", func(name string, info fs.DirEntry, err error) error {
		file_ext := name[len(name) - 3:]
		if fileExtInString(file_ext) {
			filepaths = append(filepaths, name[len("content/"):])
		}
		
		return nil
	})

	
	// Build the file tree
	tree := buildTree(filepaths)
	mds := loadMd(r.URL.Path[1:])

	page_content := mdToHTML(mds)

	current_page :=  &TestPage{FileContent: template.HTML(page_content), SideBar: *tree}

	// This enables the javascript code to work but im not 100% sure why it prints even on good requests
	if string(page_content) == "" {
		w.WriteHeader(http.StatusBadRequest)
    	w.Write([]byte("400 - Bad Request!"))
		print("Bad request!")
	} else {
		t, _ := template.ParseFiles("html/view.html")
		t.Execute(w, current_page)
	}
}

func main() {
    http.HandleFunc("/", viewHandler)
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("html/css")))) //Allow access to css folder
    http.Handle("/favicon.ico",  http.StripPrefix("/", http.FileServer(http.Dir("html/img")))) // Serve favicon
	log.Fatal(http.ListenAndServe(":8080", nil))
}
