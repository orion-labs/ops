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
                    <!--<a onClick={this.authenticate} className="btn btn-primary btn-lg btn-login btn-block">Sign In</a>-->
                </div>
            </div>
        )
    }
}

class LoggedIn extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            jokes: []
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
        // jquery fetch
        // $.get("http://localhost:3000/api/jokes", res => {
        //     this.setState({
        //         jokes: res
        //     });
        // });

        // Standard js fetch
        fetch("http://localhost:3000/api/jokes")
            .then(res => res.json())
            .then(res => {
                // asynchronous function.
                //this.setState({jokes: res})

                // this will let you log the state to the console.  logging it after this line would fail to impress
                this.setState({jokes: res}, () => {console.log(this.state)})
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
                <h2>Jokeish</h2>
                <p>Let's feed you with some funny Jokes!!!</p>
                <div className="row">
                    <div className="container">
                        {this.state.jokes.map(function(joke, i) {
                            return <Joke key={i} joke={joke} />;
                        })}
                    </div>
                </div>
            </div>
        );
    }
}

class Joke extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            liked: "",
            jokes: []
        };
        this.like = this.like.bind(this);
        this.serverRequest = this.serverRequest.bind(this);
    }

    like() {
        let joke = this.props.joke;
        this.serverRequest(joke);
    }
    serverRequest(joke) {
        $.post(
            "http://localhost:3000/api/jokes/like/" + joke.id,
            { like: 1 },
            res => {
                console.log("res... ", res);
                this.setState({ liked: "Liked!", jokes: res });
                this.props.jokes = res;
            }
        );
    }

    render() {
        return (
            <div className="col-xs-4">
                <div className="panel panel-default">
                    <div className="panel-heading">
                        #{this.props.joke.id}{" "}
                        <span className="pull-right">{this.state.liked}</span>
                    </div>
                    <div className="panel-body">{this.props.joke.joke}</div>
                    <div className="panel-footer">
                        {this.props.joke.likes} Likes &nbsp;
                        <a onClick={this.like} className="btn btn-default">
                            <span className="glyphicon glyphicon-thumbs-up" />
                        </a>
                    </div>
                </div>
            </div>
        )
    }
}


ReactDOM.render(<App />, document.getElementById('app'));