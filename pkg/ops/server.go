package ops

import (
	"crypto/tls"
	"embed"
	"fmt"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

//go:embed views
var content embed.FS

func RunServer(address string, port int) (err error) {
	router := gin.Default()

	api := router.Group("/api")

	{
		api.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "pong",
			})
		})
	}

	api.GET("/stacks", StacksHandler)
	api.GET("/stacks/:stackName", StackHandler)
	api.GET("/stacks/:stackName/ca", CaHandler)

	router.Use(Serve("/", content))

	addr := fmt.Sprintf("%s:%d", address, port)
	fmt.Printf("Server starting on %s.\n", addr)

	err = router.Run(addr)

	return err
}

func Serve(urlPrefix string, efs embed.FS) gin.HandlerFunc {
	// the embedded filesystem has a 'views/' at the top level.  We wanna strip this so we can treat the root of the views directory as the web root.
	fsys, err := fs.Sub(efs, "views")
	if err != nil {
		log.Fatalf(err.Error())
	}

	fileserver := http.FileServer(http.FS(fsys))
	if urlPrefix != "" {
		fileserver = http.StripPrefix(urlPrefix, fileserver)
	}

	return func(c *gin.Context) {
		fileserver.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}

func StacksHandler(c *gin.Context) {
	c.Header("Content-Type", "application/json")

	stacks, err := GetStacks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, make([]DisplayStack, 0))
	}

	c.JSON(http.StatusOK, stacks)
}

func StackHandler(c *gin.Context) {
	c.Header("Content-Type", "application/json")

	stackName := c.Param("stackName")
	if stackName == "" {
		c.AbortWithStatus(http.StatusNotFound)
	}

	stack, err := GetStack(stackName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, DisplayStack{})
	}

	c.JSON(http.StatusOK, stack)
}

func CaHandler(c *gin.Context) {
	stackName := c.Param("stackName")
	if stackName == "" {
		c.AbortWithStatus(http.StatusNotFound)
	}

	stack, err := GetStack(stackName)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
	}

	caUrl := stack.CA

	resp, err := http.Get(caUrl)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
	}
	defer resp.Body.Close()

	certBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
	}

	c.Data(200, "application/pkix-cert", certBytes)
}

func GetStacks() (stacks []DisplayStack, err error) {
	config, err := LoadConfig("")
	if err != nil {
		err = errors.Wrapf(err, "failed to load default config file")
		return stacks, err
	}

	s, err := NewStack(config, nil, true)
	if err != nil {
		err = errors.Wrapf(err, "Failed to create devenv object")
		return stacks, err
	}

	stacklist, err := s.ListStacks()
	if err != nil {
		err = errors.Wrapf(err, "Error listing stacks")
		return stacks, err
	}

	stacks = make([]DisplayStack, 0)

	for _, stack := range stacklist {

		display := DisplayStack{
			Name: *stack.StackName,
		}

		stacks = append(stacks, display)
	}

	return stacks, err
}

func GetStack(stackName string) (stack DisplayStack, err error) {
	config, err := LoadConfig("")
	if err != nil {
		err = errors.Wrapf(err, "failed to load default config file")
		return stack, err
	}

	config.StackName = stackName

	s, err := NewStack(config, nil, true)
	if err != nil {
		err = errors.Wrapf(err, "Failed to create devenv object")
		return stack, err
	}

	// This is a horrible hack that just gets the account from the caller - i.e. the aws creds of whomever started the server
	output, err := sts.New(s.AwsSession).GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		err = errors.Wrapf(err, "Error getting caller identity")
		return stack, err
	}

	account := *output.Account

	cfstatus, err := s.Status()
	if err != nil {
		err = errors.Wrapf(err, "failed getting status for stack %s", stackName)
		return stack, err
	}

	ctime, err := s.Created()
	if err != nil {
		err = errors.Wrapf(err, "failed getting creation time for %s", stackName)
	}

	outputs, err := s.Outputs()
	if err != nil {
		err = errors.Wrapf(err, "failed getting outputs for %s", stackName)
		return stack, err
	}

	var address string
	var caHost string
	var api string
	var login string

	for _, o := range outputs {
		switch *o.OutputKey {
		case "Address":
			address = *o.OutputValue

		case "Login":
			e := PingEndpoint(*o.OutputValue)
			if e != nil {
				login = "Not Ready"
				break
			}
			login = fmt.Sprintf("https://%s", *o.OutputValue)

		case "Api":
			e := PingEndpoint(*o.OutputValue)
			if e != nil {
				api = "Not Ready"
				break
			}
			api = fmt.Sprintf("https://%s", *o.OutputValue)

		case "CA":
			e := PingEndpoint(*o.OutputValue)
			if e != nil {
				caHost = "Not Ready"
				break
			}
			caHost = fmt.Sprintf("https://%s/v1/pki/ca/pem", *o.OutputValue)
		}
	}

	kotsadm := "Not Ready"
	e := PingEndpoint(fmt.Sprintf("%s:8800", address))
	if e == nil {
		kotsadm = fmt.Sprintf("http://%s:8800", address)
	}

	stack = DisplayStack{
		Account:  account,
		Name:     stackName,
		CFStatus: cfstatus,
		Address:  address,
		Kotsadm:  kotsadm,
		Api:      api,
		Login:    login,
		CA:       caHost,
		Created:  ctime.String(),
	}

	return stack, err
}

type DisplayStack struct {
	Account     string `json:"account" binding:"required"`
	Kubernetes  string `json:"kubernetes" binding:"required"`
	Kotsadm     string `json:"kotsadm" binding:"required"`
	CFStatus    string `json:"cfstatus" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Address     string `json:"address" binding:"required"`
	Datastore   string `json:"datastore" binding:"required"`
	EventStream string `json:"eventstream" binding:"required"`
	Media       string `json:"media" binding:"required"`
	Login       string `json:"login" binding:"required"`
	Api         string `json:"api" binding:"required"`
	CDN         string `json:"cdn" binding:"required"`
	CA          string `json:"ca" binding:"required"`
	Created     string `json:"created" binding:"required"`
}

func PingEndpoint(address string) (err error) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := http.Client{
		Timeout: time.Second,
	}

	_, err = client.Get(fmt.Sprintf("https://%s", address))

	return err
}
