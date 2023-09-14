package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"text/template"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

func main() {

	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("css"))))
	http.Handle("/image/", http.StripPrefix("/image/", http.FileServer(http.Dir("image"))))
	http.Handle("/html/", http.StripPrefix("/html/", http.FileServer(http.Dir("html"))))

	http.HandleFunc("/", accueilHandler)
	http.HandleFunc("/inscriptionCheck", inscriptionCheckHandler)
	http.HandleFunc("/connexionCheck", connexionCheckHandler)
	http.HandleFunc("/publication", publicationHandler)
	http.HandleFunc("/publicationCheck", PublicationCheckHandler)
	http.HandleFunc("/actualite", getPostsFromDB)
	http.HandleFunc("/message", MessageCheckHandler)
	http.HandleFunc("/compte", CompteCheckHandler)
	http.HandleFunc("/accueil", AccueilCheckHandler)
	http.HandleFunc("/deconnexion", Deconnexion)

	log.Println("Serveur démarré sur le port 3000...")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

type Utilisateur struct {
	Nom        string
	Email      string
	motdepasse string
}

func inscriptionCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Récupérer les données du formulaire
	utilisateur := Utilisateur{
		Nom:        r.FormValue("nom"),
		Email:      r.FormValue("email"),
		motdepasse: r.FormValue("motdepasse"),
	}

	hashMotDePasse, err := bcrypt.GenerateFromPassword([]byte(utilisateur.motdepasse), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	// Connexion à la base de données
	db, err := sql.Open("mysql", "root:root@tcp(localhost:3306)/utilisateurs")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Exécution de la requête d'insertion
	_, err = db.Exec("INSERT INTO utilisateurs (nom, email, motdepasse) VALUES (?, ?, ?)", utilisateur.Nom, utilisateur.Email, string(hashMotDePasse))
	if err != nil {
		log.Fatal(err)
	}
	http.Redirect(w, r, "/actualite", http.StatusSeeOther)
	return

}

func connexionCheckHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	motdepasse := r.FormValue("motdepasse")

	utilisateur, err := getUtilisateur(email, motdepasse)
	if err != nil {
		log.Fatal(err)
	}

	if utilisateur != nil {
		cookie := http.Cookie{
			Name:  "ID",
			Value: strconv.Itoa(utilisateur.ID),
			Expires: time.Now().Add(24 * time.Hour),
		}
		// Connexion réussie
		cookie = http.Cookie{
			Name:  "username",
			Value: utilisateur.Nom,
			Expires: time.Now().Add(24 * time.Hour),
		}
		http.SetCookie(w, &cookie)
		http.Redirect(w, r, "/actualite", http.StatusSeeOther)
		return

	} else {
		// Échec de la connexion
		fmt.Fprintf(w, "Échec de la connexion. Vérifiez votre email et votre mot de passe.")
	}
}

func connexionHandler(w http.ResponseWriter, r *http.Request) {

	var tpl *template.Template
	tpl, err := template.ParseFiles("./actualite.html")
	if err != nil {
		log.Fatal(err)
	}
	tpl.Execute(w, nil)
}

func accueilHandler(w http.ResponseWriter, r *http.Request) {
	var tpl *template.Template
	tpl, err := template.ParseFiles("./accueil.html")
	if err != nil {
		log.Fatal(err)
	}
	tpl.Execute(w, nil)
}

type Utilisateurs struct {
	ID         int
	Nom        string
	Email      string
	motdepasse string
}

func getUtilisateur(email, motdepasse string) (*Utilisateurs, error) {
	db, err := sql.Open("mysql", "root:root@tcp(localhost:3306)/utilisateurs")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := "SELECT id, nom, email, motdepasse FROM utilisateurs WHERE email = ?"
	row := db.QueryRow(query, email)

	utilisateurs := &Utilisateurs{}
	err = row.Scan(&utilisateurs.ID, &utilisateurs.Nom, &utilisateurs.Email, &utilisateurs.motdepasse)
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(utilisateurs.motdepasse), []byte(motdepasse))
	if err != nil {
		return nil, err
	}

	return utilisateurs, nil
}

func getStringPointer(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}


type Post struct {
	Nom          string
	User         Utilisateur
	ID           int
	Lieux        string
	Objet        *string
	Contenu      string
	Likes        int
	Commentaires []Commentaire
}

func publicationHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("username")
	if err != nil {
		http.Redirect(w, r, "/html/connexion.html", http.StatusSeeOther)
		return
	}

	nom := cookie.Value

	post := Post{
		Contenu: r.FormValue("contenu"),
		Lieux:   r.FormValue("lieux"),
		Objet:   getStringPointer(r.FormValue("objet")),
		Nom:     nom, // Assigner le nom de l'utilisateur au post
	}

	db, err := sql.Open("mysql", "root:root@tcp(localhost:3306)/utilisateurs")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO post (Contenu, Lieux, Objet, Nom, Likes) VALUES (?, ?, ?, ?,0)", post.Contenu, post.Lieux, post.Objet, post.Nom)
	if err != nil {
		log.Fatal(err)
	}

	http.Redirect(w, r, "/actualite", http.StatusSeeOther)
}

func getPostsFromDB(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("mysql", "root:root@tcp(localhost:3306)/utilisateurs")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT p.ID, p.lieux, p.objet, p.contenu, p.likes, Nom, c.ID, c.postID, c.auteur, c.contenu FROM post p LEFT JOIN commentaire c ON p.ID = c.postID")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var posts []Post
	postMap := make(map[int]*Post) // Utilisé pour regrouper les commentaires par post ID
	for rows.Next() {
		var post Post
		var commentaire Commentaire
		var timestamp sql.NullTime
		if err := rows.Scan(&post.ID, &post.Lieux, &post.Objet, &post.Contenu, &post.Likes, &post.Nom, &commentaire.ID, &commentaire.PostID, &commentaire.Auteur, &commentaire.Contenu); err != nil {
			log.Fatal(err)
		}
		// Assigner la valeur du timestamp à commentaire.Timestamp
		if timestamp.Valid {
			commentaire.Timestamp = sql.NullTime{Time: timestamp.Time, Valid: true}
		} else {
			commentaire.Timestamp = sql.NullTime{Valid: false}
		}
	
		if err != nil {
			log.Fatal(err)
		}
	

		if err != nil {
			log.Fatal(err)
		}
		if p, ok := postMap[post.ID]; ok {
			p.Commentaires = append(p.Commentaires, commentaire)
		
		} else {
			post.Commentaires = []Commentaire{commentaire}
			posts = append(posts, post)
			postMap[post.ID] = &posts[len(posts)-1]
		}
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	var tpl *template.Template
	tpl, err = template.ParseFiles("./actualite.html")
	if err != nil {
		log.Fatal(err)
	}
	tpl.Execute(w, posts)
}

func PublicationCheckHandler(w http.ResponseWriter, r *http.Request) {
	var tpl *template.Template
	tpl, err := template.ParseFiles("./publication.html")
	if err != nil {
		log.Fatal(err)
	}
	tpl.Execute(w, nil)

	if r.Method == "GET" {
		http.Redirect(w, r, "publication.html", http.StatusSeeOther)
		return
	}
}

// Votre code existant pour traiter la publication du formulaire...

func MessageCheckHandler(w http.ResponseWriter, r *http.Request) {
	var tpl *template.Template
	tpl, err := template.ParseFiles("./message.html")
	if err != nil {
		log.Fatal(err)
	}
	tpl.Execute(w, nil)

	if r.Method == "GET" {
		http.Redirect(w, r, "message.html", http.StatusSeeOther)
		return
	}
}

func AccueilCheckHandler(w http.ResponseWriter, r *http.Request) {
	var tpl *template.Template
	tpl, err := template.ParseFiles("./accueil.html")
	if err != nil {
		log.Fatal(err)
	}
	tpl.Execute(w, nil)

	if r.Method == "GET" {
		http.Redirect(w, r, "accueil.html", http.StatusSeeOther)
		return
	}
}

func CompteCheckHandler(w http.ResponseWriter, r *http.Request) {
	var tpl *template.Template
	tpl, err := template.ParseFiles("./compte.html")
	if err != nil {
		log.Fatal(err)
	}
	tpl.Execute(w, nil)

	if r.Method == "GET" {
		http.Redirect(w, r, "compte.html", http.StatusSeeOther)
		return
	}
}

func Deconnexion(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{
		Name:   "ID",
		Value:  "",
		MaxAge: -1, // Définissez MaxAge sur -1 pour supprimer le cookie
	}
	http.SetCookie(w, cookie)

	cookie = &http.Cookie{
		Name:   "username",
		Value:  "",
		MaxAge: -1, // Définissez MaxAge sur -1 pour supprimer le cookie
	}
	http.SetCookie(w, cookie)

	// Redirigez vers la page de connexion ou toute autre localisation souhaitée
	http.Redirect(w, r, "/accueil", http.StatusSeeOther)
}

func CommentaireHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusBadRequest)
		return
	}

	postID := r.FormValue("postID")
	auteur := r.FormValue("auteur")
	contenu := r.FormValue("contenu")

	db, err := sql.Open("mysql", "root:root@tcp(localhost:3306)/utilisateurs")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO commentaire (postID, auteur, contenu, timestamp) VALUES (?, ?, ?, NOW())", postID, auteur, contenu)

	if err != nil {
		log.Fatal(err)
	}

	http.Redirect(w, r, "/actualite", http.StatusSeeOther)
}

func LikeHandler(w http.ResponseWriter, r *http.Request) {
	postID := r.FormValue("postID")
	if postID == "" {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	db, err := sql.Open("mysql", "root:root@tcp(localhost:3306)/utilisateurs")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	query := "UPDATE post SET likes = likes + 1 WHERE id = ?"
	_, err = db.Exec(query, postID)
	if err != nil {
		log.Fatal(err)
	}

	http.Redirect(w, r, "/actualite", http.StatusSeeOther)
}

func init() {
	http.HandleFunc("/like", LikeHandler)
	http.HandleFunc("/commentaire", CommentaireHandler)
}

type Commentaire struct {
	ID        sql.NullInt64
	PostID    sql.NullInt64
	Auteur    sql.NullString
	Contenu   sql.NullString
	Timestamp sql.NullTime
}

