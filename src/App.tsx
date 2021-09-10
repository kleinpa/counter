import React, { useEffect } from "react";
import {
  BrowserRouter as Router,
  Switch,
  Route,
  Link,
  RouteComponentProps,
  Redirect,
} from "react-router-dom";

import "./App.scss";

import {
  IncrementRequest,
  SetRequest,
  WatchReply,
  WatchRequest,
} from "./counter_pb";
import { CounterClient } from "./CounterServiceClientPb";

export class CounterBackend {
  readonly be: CounterServerBackend;
  readonly id: string;

  constructor(be: CounterServerBackend, id: string) {
    this.id = id;
    this.be = be;
  }

  increment(x = 1): Promise<number> {
    return new Promise((resolve, reject) => {
      const req = new IncrementRequest();
      req.setValue(x);
      req.setId(this.id);
      this.be.client.increment(req, {}, (err, res) => {
        if (err) {
          reject(err.message);
        } else {
          resolve(res.getValue());
        }
      });
    });
  }

  watch(update: (res: number) => void, cancelled: Promise<void>): void {
    const req = new WatchRequest();
    req.setId(this.id);
    (async () => {
      let done = false;
      cancelled = cancelled.then(() => {
        done = true;
      });
      while (!done) {
        try {
          // Wrap streaming response in a Promise so we can await it and retry.
          await new Promise<void>(
            /* eslint-disable-line no-loop-func */ (resolve, reject) => {
              const stream = this.be.client.watch(req, {});
              stream.on("data", (resu) => {
                const res = resu as WatchReply;
                update(res.getValue());
              });
              stream.on("error", (error) => {
                stream.cancel();
                reject(error);
              });
              stream.on("end", () => {
                stream.cancel();
                resolve();
              });
              cancelled.then(() => {
                done = true;
                stream.cancel();
                resolve();
              });
            }
          );
        } catch (error) {
          await new Promise((r) => setTimeout(r, 300));
        }
      }
    })();
  }

  set(x = 0): Promise<void> {
    return new Promise((resolve, reject) => {
      const req = new SetRequest();
      req.setValue(x);
      req.setId(this.id);
      this.be.client.set(req, {}, (err) => {
        if (err) {
          reject(err.message);
        } else {
          resolve();
        }
      });
    });
  }
}

export class CounterServerBackend {
  readonly client: CounterClient;

  constructor() {
    this.client = new CounterClient(process.env.REACT_APP_API_URL || "/api");
  }

  counter(id: string): CounterBackend {
    return new CounterBackend(this, id);
  }
}

interface CounterDisplayProps {
  counter: CounterBackend;
}
function CounterDisplay(props: CounterDisplayProps) {
  const [value, setValue] = React.useState(0);

  let cancel: () => void;
  useEffect(() => {
    const cancelled = new Promise<void>((resolve) => (cancel = resolve));
    props.counter.watch((value) => {
      setValue(value);
    }, cancelled);

    return () => {
      cancel();
    };
  }, []);

  return (
    <div className="CountControl">
      <div className="count">{value}</div>
    </div>
  );
}

interface CounterControlProps {
  counter: CounterBackend;
}
function CounterControl(props: CounterControlProps) {
  function increment() {
    props.counter.increment();
  }
  function zero() {
    props.counter.set(0);
  }

  return (
    <div className="CountControl">
      <button onClick={increment}>+</button>
      <button onClick={zero}>0</button>
    </div>
  );
}

export default function App(): JSX.Element {
  const be = new CounterServerBackend();

  function NewRedirect() {
    useEffect(() => {
      be.counter(x).set(0);
    });
    const x = Math.random().toString(36).slice(2);

    return <Redirect to={`/${x}/control`} />;
  }

  interface CounterControlParams {
    id: string;
  }
  function CounterControlRoute(
    props: RouteComponentProps<CounterControlParams>
  ) {
    return (
      <div>
        <CounterDisplay counter={be.counter(props.match.params.id)} />
        <CounterControl counter={be.counter(props.match.params.id)} />
        <Link to="/">back</Link>{" "}
        <Link to={`/${props.match.params.id}`}>view</Link>
      </div>
    );
  }

  interface CounterDisplayParams {
    id: string;
  }
  function CounterDisplayRoute(
    props: RouteComponentProps<CounterDisplayParams>
  ) {
    return (
      <div>
        <CounterDisplay counter={be.counter(props.match.params.id)} />
      </div>
    );
  }
  return (
    <Router>
      <div className="App">
        <Switch>
          <Route path="/" exact={true}>
            {process.env.REACT_APP_NAME} {process.env.REACT_APP_VERSION}{" "}
            <Link to="/new">new</Link>
          </Route>
          <Route path="/new">
            <NewRedirect />
          </Route>
          <Route path="/:id/control" component={CounterControlRoute} />
          <Route path="/:id" component={CounterDisplayRoute} />
        </Switch>
      </div>
    </Router>
  );
}
