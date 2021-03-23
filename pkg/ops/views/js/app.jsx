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
            stackDetails: {},
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

    getStackDetails(name) {
        fetch(`http://localhost:3000/api/stacks/name/${name}`
        )
        .then( res => JSON.stringify(res))
        .then( jsonResults => {
            let tempDetails = this.state.stackDetails
            tempDetails[name] = jsonResults
            this.setState({ stackDetails: tempDetails })
        })
    }

    async componentDidMount() {
        await this.serverRequest()
        this.state.stacks.forEach( stack => {
            this.getStackDetails(stack.name)
        })
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
                            return <Stack
                                key={`stack-${stack.name}`}
                                stack={stack}
                                stackDetails={this.state.stackDetails[stack.name]}
                            />;
                        })}
                    </div>
                </div>
            </div>
        );
    }
}

function Stack(props) {
    return (
        <div className="col-lg-6">
            <div className="panel panel-default">
                <div className="panel-heading">
                    {props.stack.name}{" "}
                </div>
                <div className="panel-body">
                    Created: {props.stackDetails.created}<br/>
                    Address: {props.stackDetails.address}<br/>
                    Account: {props.stackDetails.account}<br/>
                    CloudFormation: {props.stackDetails.cfstatus}<br/>
                    Kotsadm: <a href={props.stackDetails.kotsadm}>{props.stackDetails.kotsadm}</a> <br/>
                    Login: <a href={props.stackDetails.login}>{props.stackDetails.login}</a><br/>
                    API: <a href={props.stackDetails.api}>{props.stackDetails.api}</a><br/>
                    CA: <a href={props.stackDetails.ca}>{props.stackDetails.ca}</a><br/>
                </div>
                <div className="panel-footer">
                </div>
            </div>
        </div>
    )
}


ReactDOM.render(<App />, document.getElementById('app'));
