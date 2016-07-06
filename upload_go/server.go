package main

import (
	"fmt"
	"gopkg.in/mgo.v2"
	"time"
	//  "gopkg.in/mgo.v2/bson"
	"html/template"
	"io"
	//	"log"
	"crypto/md5"
	"net/http"
	//  "os"
	//	"mime"
)

type Document struct {
	Filename string
}

func upload(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		h := md5.New()
		token := fmt.Sprintf("%x", h.Sum(nil))
		t, _ := template.ParseFiles("upload.html")
		t.Execute(w, token)
	} else {
		// on stocke les fichiers de -64mo en ram
		// connexion à la bdd
		session, err := mgo.Dial("localhost")
		if err != nil {
			panic(err)
		}
		defer session.Close()
		// test gridFS
		r.ParseMultipartForm(16 << 20)
		gridfs := session.DB("miogo").GridFS("fs")
		file, handler, err := r.FormFile("uploadfile")
		if err != nil {
			panic(err)
		}
		gridfsFile, err := gridfs.Create(handler.Filename)
		started := time.Now()
		if err != nil {
			panic(err)
		}
		defer gridfsFile.Close()
		defer file.Close()
		io.Copy(gridfsFile, file)
		fmt.Printf("%s secondes\n", time.Since(started))
	}
}

func download(w http.ResponseWriter, r *http.Request) {
	session, _ := mgo.Dial("localhost")
	var results []Document
	session.DB("miogo").GridFS("fs").Find(nil).All(&results)
	fmt.Println("resultats :", results)
	t, _ := template.ParseFiles("download.html")
	t.Execute(w, results)
	//db.fs.files.find({}, { filename: 1, _id:0 })
	//file, _ := session.DB("miogo").GridFS("fs").Open("fourni.jpg")
	//io.Copy(w, file)
}

func downloadFile(w http.ResponseWriter, r *http.Request) {
	session, _ := mgo.Dial("localhost")
	fileName := r.URL.Path[len("/download/"):]
	fmt.Println(fileName)
	file, _ := session.DB("miogo").GridFS("fs").Open(fileName)
	io.Copy(w, file)
}

func main() {
	http.HandleFunc("/upload", upload)
	http.HandleFunc("/download", download)
	http.HandleFunc("/download/", downloadFile)
	http.ListenAndServe(":9090", nil)
}
