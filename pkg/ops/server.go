package ops

import (
	"crypto/tls"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed views
var content embed.FS

type OnpremDetails struct {
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
	Uptime      string `json:"uptime" binding:"required"`
}

type Account struct {
	Number    string `json:"account_number"`
	KeyId     string `json:"aws_access_key_id"`
	SecretKey string `json:"aws_secret_access_key"`
	Region    string `json:"aws_region"`
}

type OpsServer struct {
	Address  string
	Port     int
	Accounts []Account
}

const ACCOUNT_ENV_VAR = "AWS_ACCOUNT_CREDENTIALS"

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.DebugLevel)
}

func NewOpsServer(address string, port int) (server *OpsServer, err error) {
	accounts := make([]Account, 0)

	if os.Getenv(ACCOUNT_ENV_VAR) != "" {
		decoded, err := base64.StdEncoding.DecodeString(os.Getenv(ACCOUNT_ENV_VAR))
		if err != nil {
			err = errors.Wrapf(err, "failed to decode base64 encoded creds from environment")
			return server, err
		}

		err = json.Unmarshal(decoded, &accounts)
		if err != nil {
			err = errors.Wrapf(err, "Failed unmarshalling json in %s", ACCOUNT_ENV_VAR)
			return server, err
		}

		log.Debugf("Using Credentials from %s", ACCOUNT_ENV_VAR)

	} else {
		log.Debugf("Using Default Credentials")
		sess, err := DefaultSession()
		if err != nil {
			log.Fatalf("failed creating aws session: %s", err)
		}

		// This is a horrible hack that just gets the account from the caller - i.e. the aws creds of whomever started the server
		output, err := sts.New(sess).GetCallerIdentity(&sts.GetCallerIdentityInput{})
		if err != nil {
			err = errors.Wrapf(err, "Error getting caller identity")
			return server, err
		}

		account := Account{
			Number: *output.Account,
		}

		accounts = append(accounts, account)
	}

	server = &OpsServer{
		Address:  address,
		Port:     port,
		Accounts: accounts,
	}

	return server, err
}

func (s *OpsServer) Run() (err error) {
	router := gin.Default()

	api := router.Group("/api")

	{
		api.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "pong",
			})
		})
	}

	api.GET("/stacks", s.InstancesHandler)
	api.GET("/stacks/:account/:stackName", s.SingleInstanceHandler)
	api.GET("/stacks/:account/:stackName/ca", s.InstanceCaHandler)
	api.DELETE("/stacks/:account/:stackName", s.InstanceDeleteHandler)

	router.Use(s.Serve("/", content))

	addr := fmt.Sprintf("%s:%d", s.Address, s.Port)
	fmt.Printf("Server starting on %s.\n", addr)

	err = router.Run(addr)

	return err
}

func (s *OpsServer) Serve(urlPrefix string, efs embed.FS) gin.HandlerFunc {
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

// InstancesHandler returns json with all instances, though the instances themselves will only have account numbers and names.  Details need to be fetched later.  This is done to speed response time on the webpage.
func (s *OpsServer) InstancesHandler(c *gin.Context) {
	c.Header("Content-Type", "application/json")

	instances, err := s.GetInstances()
	if err != nil {
		c.JSON(http.StatusInternalServerError, make([]OnpremDetails, 0))
		log.Errorf("Error in instances handler: %s", err)
		return
	}

	c.JSON(http.StatusOK, instances)
}

// SingleInstanceHandler returns details for a particular instance.
func (s *OpsServer) SingleInstanceHandler(c *gin.Context) {
	c.Header("Content-Type", "application/json")

	account := c.Param("account")
	if account == "" {
		c.AbortWithStatus(http.StatusNotFound)
	}

	stackName := c.Param("stackName")
	if stackName == "" {
		c.AbortWithStatus(http.StatusNotFound)
	}

	deets, err := s.GetDetails(account, stackName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, OnpremDetails{})
		log.Errorf("Error in single instance handler: %s", err)
		return
	}

	c.JSON(http.StatusOK, deets)
}

// InstanceCaHandler fetches the CA cert for a specific instance and sends it back to the client.
func (s *OpsServer) InstanceCaHandler(c *gin.Context) {
	account := c.Param("account")
	if account == "" {
		c.AbortWithStatus(http.StatusNotFound)
	}

	stackName := c.Param("stackName")
	if stackName == "" {
		c.AbortWithStatus(http.StatusNotFound)
	}

	stack, err := s.GetDetails(account, stackName)
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

// InstanceDeleteHandler deletes a specific instance
func (s *OpsServer) InstanceDeleteHandler(c *gin.Context) {
	account := c.Param("account")
	if account == "" {
		c.AbortWithStatus(http.StatusNotFound)
	}

	stackName := c.Param("stackName")
	if stackName == "" {
		c.AbortWithStatus(http.StatusNotFound)
	}

	err := s.DeleteStack(account, stackName)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
	}

	c.Status(http.StatusOK)
}

// GetStack retrieves a configured stack object for a given account and name based on credentials we have available.
func (s *OpsServer) GetStack(accountNumber string, stackName string) (stack *Stack, err error) {
	log.Debugf("Generating Stack object for account: %q name: %q", accountNumber, stackName)

	var account *Account

	for _, a := range s.Accounts {
		log.Debugf("Checking %s against %s", a.Number, accountNumber)
		if a.Number == accountNumber {
			log.Debugf("Setting accoutn as %s", a.Number)
			account = &a
			break
		}
	}

	if account == nil {
		err = errors.New(fmt.Sprintf("Failed to retrieve Account Object for %q", accountNumber))
		return stack, err
	}

	log.Debugf("Account is %s", account.Number)

	config := StackConfig{
		StackName: stackName,
	}

	var awsSession *session.Session

	// If we haven't done any special account and credential provisioning, get them in the normal fashion
	if account.KeyId == "" && account.SecretKey == "" {
		log.Debugf("Using DefaultSession")
		awsSession, err = DefaultSession()
		if err != nil {
			err = errors.Wrapf(err, "failed to get default session")
			return stack, err
		}
	} else { // otherwise, use what was explicitly provisioned
		log.Debugf("Creating Session from static creds.  ID: %s", account.KeyId)
		awsSession, err = session.NewSession(&aws.Config{
			Region:      aws.String(account.Region),
			Credentials: credentials.NewStaticCredentials(account.KeyId, account.SecretKey, ""),
		})
		if err != nil {
			err = errors.Wrapf(err, "failed to create session from static creds")
			return stack, err
		}
	}

	output, err := sts.New(awsSession).GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		err = errors.Wrapf(err, "Error getting caller identity")
		return stack, err
	}

	log.Debugf("getting stack from account %s", *output.Account)

	stack, err = NewStack(&config, awsSession, false)
	if err != nil {
		err = errors.Wrapf(err, "failed creating stack object")
		return stack, err
	}

	return stack, err
}

func (s *OpsServer) DeleteStack(accountNumber string, stackName string) (err error) {
	stack, err := s.GetStack(accountNumber, stackName)
	if err != nil {
		err = errors.Wrapf(err, "failed to generate stack for account %s name %s", accountNumber, stackName)
		return err
	}

	err = stack.Delete()

	return err
}

func (s *OpsServer) GetInstances() (instances []OnpremDetails, err error) {
	instances = make([]OnpremDetails, 0)

	for _, account := range s.Accounts {
		log.Debugf("Getting stacks for %s", account.Number)
		stack, err := s.GetStack(account.Number, "")
		if err != nil {
			err = errors.Wrapf(err, "failed to generate stack for account %s", account.Number)
			return instances, err
		}

		stacklist, err := stack.ListStacks()
		if err != nil {
			err = errors.Wrapf(err, "error listing stacks")
			return instances, err
		}

		log.Debugf("%d instances for %s", len(stacklist), account.Number)

		for _, stack := range stacklist {

			display := OnpremDetails{
				Name:    *stack.StackName,
				Account: account.Number,
			}

			instances = append(instances, display)
		}

	}

	return instances, err
}

func (s *OpsServer) GetDetails(accountNum string, stackName string) (deets OnpremDetails, err error) {
	stack, err := s.GetStack(accountNum, stackName)
	if err != nil {
		err = errors.Wrapf(err, "failed to generate stack for account %s name %s", accountNum, stackName)
		return deets, err
	}

	cfstatus, err := stack.Status()
	if err != nil {
		err = errors.Wrapf(err, "failed getting status for stack %s", stackName)
		return deets, err
	}

	ctime, err := stack.Created()
	if err != nil {
		err = errors.Wrapf(err, "failed getting creation time for %s", stackName)
	}

	outputs, err := stack.Outputs()
	if err != nil {
		err = errors.Wrapf(err, "failed getting outputs for %s", stackName)
		return deets, err
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

	dur := time.Now().Sub(*ctime)

	uptime := dur.String()

	// very crude formatting
	uptime = strings.ReplaceAll(uptime, "h", "h ")
	uptime = strings.ReplaceAll(uptime, "m", "m ")

	deets = OnpremDetails{
		Account:  accountNum,
		Name:     stackName,
		CFStatus: cfstatus,
		Address:  address,
		Kotsadm:  kotsadm,
		Api:      api,
		Login:    login,
		CA:       caHost,
		Created:  ctime.String(),
		Uptime:   uptime,
	}

	return deets, err
}

func PingEndpoint(address string) (err error) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := http.Client{
		Timeout: time.Second,
	}

	_, err = client.Get(fmt.Sprintf("https://%s", address))

	return err
}
