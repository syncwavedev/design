package main

import (
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/public/", public)
	mux.HandleFunc("/", pageHandler)
	log.Printf("Development server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func public(w http.ResponseWriter, r *http.Request) {
	// Serve static files from the public directory with cache disabled
	fs := http.FileServer(http.Dir("public"))
	prefix := http.StripPrefix("/public/", fs)
	w.Header().Set("Cache-Control", "no-store")
	prefix.ServeHTTP(w, r)
}

func pageHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index"
	}
	path = strings.TrimPrefix(path, "/")

	// Disallow direct access to assets
	if strings.HasPrefix(path, "assets/") {
		http.NotFound(w, r)
		return
	}
	w.Header().Add("Cache-Control", "no-store")

	// Create a new master template.
	tmpl := template.New("master").Funcs(template.FuncMap{
		"seq": seq,
	})

	// Load all shared templates recursively.
	if err := loadTemplates(tmpl, "shared"); err != nil {
		log.Printf("Error loading shared templates: %v", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	// Load the page template.
	pagePath := filepath.Join("pages", path+".html")
	if _, err := os.Stat(pagePath); err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		log.Printf("stat %s: %v", pagePath, err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	if err := loadTemplateFile(tmpl, pagePath, "pages"); err != nil {
		log.Printf("parse page %v: %v", pagePath, err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	// Execute the page template by its relative name.
	// We use the base name of the page file relative to "pages" directory.
	pageName := filepath.Base(pagePath)
	if err := tmpl.ExecuteTemplate(w, pageName, nil); err != nil {
		log.Printf("execute %s: %v", pagePath, err)
	}
}

// loadTemplates walks through a directory (like "shared") and loads all .html files
// into the master template using their path relative to baseDir as the template name.
func loadTemplates(t *template.Template, baseDir string) error {
	return filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".html") {
			return loadTemplateFile(t, path, baseDir)
		}
		return nil
	})
}

// loadTemplateFile reads a file and adds it to the template t.
// The template name is the file's path relative to baseDir.
func loadTemplateFile(t *template.Template, fullPath, baseDir string) error {
	rel, err := filepath.Rel(baseDir, fullPath)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}
	// Create a new template with the relative path as its name.
	_, err = t.New(rel).Parse(string(content))
	return err
}

// seq returns a slice of integers from 0 to n-1. Useful in templates.
func seq(n int) []int {
	result := make([]int, n)
	for i := 0; i < n; i++ {
		result[i] = i
	}
	return result
}
