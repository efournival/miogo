<html>
	<head>
		<meta charset="utf-8" />
		<title>Miogo</title>
	</head>
	<body>
		<h2>Dossier {{.Path}}</h2>
		{{range .Folders}}
			<p><a href="/view{{.Path}}">{{.Path}}/</a></p>
		{{end}}
		{{range .Files}}
			<p><a href="/view{{$.Path}}/{{.Name}}">{{.Name}}</a></p>
		{{end}}
		<h2>Upload</h2>
		<form enctype="multipart/form-data" action="/upload" method="post">
			<input type="file" name="file" multiple="multiple" />
			<input type="hidden" name="path" value="{{.Path}}" />
			<input type="submit" value="upload" />
		</form>
		<h2>Nouveau dossier</h2>
		<form action="/newFolder" method="post">
			<input type="text" name="folderName" />
			<input type="hidden" name="path" value="{{.Path}}" />
			<input type="submit" value="créer" />
		</form>
	</body>
</html>
