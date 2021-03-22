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
            stacks: []
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
        fetch("http://localhost:3000/api/systems")
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
        this.serverRequest();
    }

    render() {
        return (
            <div className="container">
                <br />
                <span className="pull-right">
          <a onClick={this.logout}>Log out</a>
        </span>
                <h2>Orion PTT System Instances</h2>
                <p></p>
                <div className="row">
                    <div className="container">
                        {this.state.stacks.map(function(stack, i) {
                            return <Stack key={i} stack={stack} />;
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
            liked: "",
            stacks: []
        };
        this.like = this.like.bind(this);
        this.serverRequest = this.serverRequest.bind(this);
    }

    like() {
        let stack = this.props.stack;
        this.serverRequest(stack);
    }
    serverRequest(stack) {
        $.post(
            "http://localhost:3000/api/stacks/status`/" + stack.name,
            { like: 1 },
            res => {
                console.log("res... ", res);
                this.setState({ liked: "Liked!", stacks: res });
                this.props.stacks = res;
            }
        );
    }

    render() {
        return (
            <div className="col-lg-6">
                <div className="panel panel-default">
                    <div className="panel-heading">
                        {this.props.stack.name}{" "}
                    </div>
                    <div className="panel-body">
                        Created: {this.props.stack.created}<br/>
                        Address: {this.props.stack.address}<br/>
                        Account: {this.props.stack.account}<br/>
                        CloudFormation: {this.props.stack.cfstatus}<br/>
                        Kotsadm: <a href={this.props.stack.kotsadm}>{this.props.stack.kotsadm}</a> <br/>
                        Login: <a href={this.props.stack.login}>{this.props.stack.login}</a><br/>
                        API: <a href={this.props.stack.api}>{this.props.stack.api}</a><br/>
                        CA: <a href={this.props.stack.ca}>{this.props.stack.ca}</a><br/>
                    </div>
                    <div className="panel-footer">
                    </div>
                </div>
            </div>
        )
    }
}


ReactDOM.render(<App />, document.getElementById('app'));