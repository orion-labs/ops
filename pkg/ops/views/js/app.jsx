class App extends React.Component {
    render() {
        // if (this.loggedIn) {
            return (<LoggedIn />);
        // } else {
        //     return (<Home />);
        // }
    }
}

class Home extends React.Component {
    render() {
        return (
            <div className="container">
                <div className="col-xs-8 col-xs-offset-2 jumbotron text-center">
                    <h1>Orion PTT Systems</h1>
                    <p>Private Enviornments for Development and Testing</p>
                    <a onClick={this.authenticate} className="btn btn-primary btn-lg btn-login btn-block">Sign In</a>
                </div>
            </div>
        )
    }
}

class LoggedIn extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            stacks: [],
        };

        this.serverRequest = this.serverRequest.bind(this);
        this.logout = this.logout.bind(this);
    }

    logout() {
        localStorage.removeItem("id_token");
        localStorage.removeItem("access_token");
        localStorage.removeItem("profile");
        location.reload();
    }

    serverRequest() {
        return fetch(window.location.href + "api/stacks")
            .then(res => res.json())
            .then(res => {
                // asynchronous function.
                //this.setState({stacks: res})

                // this will let you log the state to the console.  logging it after this line would fail to impress
                this.setState({stacks: res}, () => {console.log(this.state)})
            })
            .catch(err => {console.log("ahhhhhh!", err)})
    }

    componentDidMount() {
        this.serverRequest()
        window.setInterval(this.serverRequest, 30000)
    }

    render() {
        return (
            <div className="container">
                <br />
                <span className="pull-right">
                    {/*<a onClick={this.logout}>Log out</a>*/}
                </span>
                <h2>Orion PTT System Instances</h2>
                <p></p>
                <div className="row">
                    <div className="container">
                        {this.state.stacks.map(function(stack, i) {
                            return <Stack
                                key={`stack-${stack.name}`}
                                stack={stack}
                            />;
                        })}
                    </div>
                </div>
            </div>
        );
    }
}

class Stack extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            stack: {name: '', created: '', address: '', account: '', cfstatus: '', kotsadm: '', login: '', api: '', ca: ''},
        };
    }

    destroyStack = (name) => {
        if (window.confirm("Destroy Stack " + name + "?")){
            fetch(window.location.href + `api/stacks/${name}`, {method: 'DELETE'})
        }
    }

    componentDidMount() {
        this.getStackDetails()
        var intervalID = window.setInterval(this.getStackDetails, 30000)
    }

    getStackDetails = () => {
        const { name } = this.props.stack
        fetch(window.location.href + `api/stacks/${name}`
        )
        .then( res => res.json())
        .then( jsonResults => {
            this.setState({ stack: jsonResults })
        })
    }

    render() {
        return (
            <div className="col-lg-6">
                <div className="panel panel-default">
                    <div className="panel-heading">
                        {this.props.stack.name}
                    </div>
                    <div className="panel-body">
                        Created: {this.state.stack.created}<br/>
                        Address: {this.state.stack.address}<br/>
                        Account: {this.state.stack.account}<br/>
                        CloudFormation: {this.state.stack.cfstatus}<br/>
                        Kotsadm: <a href={this.state.stack.kotsadm}>{this.state.stack.kotsadm}</a> <br/>
                        Login: <a href={this.state.stack.login}>{this.state.stack.login}</a><br/>
                        API: <a href={this.state.stack.api}>{this.state.stack.api}</a><br/>
                        CA: <a download={`CA-${this.state.stack.name}.pem`} href={window.location.href + `api/stacks/${this.state.stack.name}/ca`}>{this.state.stack.ca}</a><br/>
                    </div>
                    <div className="panel-footer">
                        <button type="button" class="btn btn-dark" onClick={() => {this.destroyStack(this.state.stack.name)}}> Destroy </button>
                    </div>
                </div>
            </div>
        )
    }
}


ReactDOM.render(<App />, document.getElementById('app'));
