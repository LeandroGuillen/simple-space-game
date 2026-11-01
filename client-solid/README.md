## Local server ports

The Go game server defaults to `http://127.0.0.1:8080`; enter this address in the setup screen. The separate AI manager defaults to port `8090`.

## Usage

```bash
$ npm install # or pnpm install or yarn install
```

### Learn more on the [Solid Website](https://solidjs.com) and come chat with us on our [Discord](https://discord.com/invite/solidjs)

## Available Scripts

In the project directory, you can run:

### `npm run dev`

Runs the app in the development mode.<br>
Open [http://localhost:5173](http://localhost:5173) to view it in the browser.

### `npm run build`

Builds the app for production to the `dist` folder.<br>
It correctly bundles Solid in production mode and optimizes the build for the best performance.

The build is minified and the filenames include the hashes.<br>
Your app is ready to be deployed!

## Saved games

Full-state downloads are available for games with one player slot. Downloads for multi-slot games are disabled to protect hidden player state. The regular game endpoint remains fog-of-war filtered; upload the downloaded JSON to restore the full game.

## Deployment

Learn more about deploying your application with the [documentations](https://vite.dev/guide/static-deploy.html)
