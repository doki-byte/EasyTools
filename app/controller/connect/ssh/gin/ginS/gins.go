// Copyright 2014 Manu Martinez-Almeida. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package ginS

import (
	gin2 "EasyTools/app/controller/connect/ssh/gin"
	"html/template"
	"net/http"
	"sync"
)

var once sync.Once
var internalEngine *gin2.Engine

func engine() *gin2.Engine {
	once.Do(func() {
		internalEngine = gin2.Default()
	})
	return internalEngine
}

// LoadHTMLGlob is a wrapper for Engine.LoadHTMLGlob.
func LoadHTMLGlob(pattern string) {
	engine().LoadHTMLGlob(pattern)
}

// LoadHTMLFiles is a wrapper for Engine.LoadHTMLFiles.
func LoadHTMLFiles(files ...string) {
	engine().LoadHTMLFiles(files...)
}

// SetHTMLTemplate is a wrapper for Engine.SetHTMLTemplate.
func SetHTMLTemplate(templ *template.Template) {
	engine().SetHTMLTemplate(templ)
}

// NoRoute adds handlers for NoRoute. It returns a 404 code by default.
func NoRoute(handlers ...gin2.HandlerFunc) {
	engine().NoRoute(handlers...)
}

// NoMethod is a wrapper for Engine.NoMethod.
func NoMethod(handlers ...gin2.HandlerFunc) {
	engine().NoMethod(handlers...)
}

// Group creates a new router group. You should add all the routes that have common middlewares or the same path prefix.
// For example, all the routes that use a common middleware for authorization could be grouped.
func Group(relativePath string, handlers ...gin2.HandlerFunc) *gin2.RouterGroup {
	return engine().Group(relativePath, handlers...)
}

// Handle is a wrapper for Engine.Handle.
func Handle(httpMethod, relativePath string, handlers ...gin2.HandlerFunc) gin2.IRoutes {
	return engine().Handle(httpMethod, relativePath, handlers...)
}

// POST is a shortcut for router.Handle("POST", path, handle)
func POST(relativePath string, handlers ...gin2.HandlerFunc) gin2.IRoutes {
	return engine().POST(relativePath, handlers...)
}

// GET is a shortcut for router.Handle("GET", path, handle)
func GET(relativePath string, handlers ...gin2.HandlerFunc) gin2.IRoutes {
	return engine().GET(relativePath, handlers...)
}

// DELETE is a shortcut for router.Handle("DELETE", path, handle)
func DELETE(relativePath string, handlers ...gin2.HandlerFunc) gin2.IRoutes {
	return engine().DELETE(relativePath, handlers...)
}

// PATCH is a shortcut for router.Handle("PATCH", path, handle)
func PATCH(relativePath string, handlers ...gin2.HandlerFunc) gin2.IRoutes {
	return engine().PATCH(relativePath, handlers...)
}

// PUT is a shortcut for router.Handle("PUT", path, handle)
func PUT(relativePath string, handlers ...gin2.HandlerFunc) gin2.IRoutes {
	return engine().PUT(relativePath, handlers...)
}

// OPTIONS is a shortcut for router.Handle("OPTIONS", path, handle)
func OPTIONS(relativePath string, handlers ...gin2.HandlerFunc) gin2.IRoutes {
	return engine().OPTIONS(relativePath, handlers...)
}

// HEAD is a shortcut for router.Handle("HEAD", path, handle)
func HEAD(relativePath string, handlers ...gin2.HandlerFunc) gin2.IRoutes {
	return engine().HEAD(relativePath, handlers...)
}

// Any is a wrapper for Engine.Any.
func Any(relativePath string, handlers ...gin2.HandlerFunc) gin2.IRoutes {
	return engine().Any(relativePath, handlers...)
}

// StaticFile is a wrapper for Engine.StaticFile.
func StaticFile(relativePath, filepath string) gin2.IRoutes {
	return engine().StaticFile(relativePath, filepath)
}

// Static serves files from the given file system root.
// Internally a http.FileServer is used, therefore http.NotFound is used instead
// of the Router's NotFound handler.
// To use the operating system's file system implementation,
// use :
//
//	router.Static("/static", "/var/www")
func Static(relativePath, root string) gin2.IRoutes {
	return engine().Static(relativePath, root)
}

// StaticFS is a wrapper for Engine.StaticFS.
func StaticFS(relativePath string, fs http.FileSystem) gin2.IRoutes {
	return engine().StaticFS(relativePath, fs)
}

// Use attaches a global middleware to the router. i.e. the middlewares attached through Use() will be
// included in the handlers chain for every single request. Even 404, 405, static files...
// For example, this is the right place for a logger or error management middleware.
func Use(middlewares ...gin2.HandlerFunc) gin2.IRoutes {
	return engine().Use(middlewares...)
}

// Routes returns a slice of registered routes.
func Routes() gin2.RoutesInfo {
	return engine().Routes()
}

// Run attaches to a http.Server and starts listening and serving HTTP requests.
// It is a shortcut for http.ListenAndServe(addr, router)
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func Run(addr ...string) (err error) {
	return engine().Run(addr...)
}

// RunTLS attaches to a http.Server and starts listening and serving HTTPS requests.
// It is a shortcut for http.ListenAndServeTLS(addr, certFile, keyFile, router)
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func RunTLS(addr, certFile, keyFile string) (err error) {
	return engine().RunTLS(addr, certFile, keyFile)
}

// RunUnix attaches to a http.Server and starts listening and serving HTTP requests
// through the specified unix socket (i.e. a file)
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func RunUnix(file string) (err error) {
	return engine().RunUnix(file)
}

// RunFd attaches the router to a http.Server and starts listening and serving HTTP requests
// through the specified file descriptor.
// Note: the method will block the calling goroutine indefinitely unless on error happens.
func RunFd(fd int) (err error) {
	return engine().RunFd(fd)
}
