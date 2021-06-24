# linkding-cleaner

A command line application that scans your bookmarks and archives dead links.

***What is Linkding?**: Linkding is a self-hosted bookmark service like Pinboard and Delicious. Only private because it is self-hosted.*

## Install

Download repository and run `make build` to compile.

### Configuration

Download [config.example.yml](https://github.com/dakotahp/linkding-cleaner/blob/master/config.example.yml) and save it as `config.yml`. Enter your domain used for your Linkding install in the `base_url` value. Then, enter your API Token from Linkding in the `api_token` value. This is found at `youdomain.com/settings/api` in your Linkding app.

Put the config file either in the same place as the app or in `$HOME/.linkdig-cleaner/config.yml`.

## To Do

- Handle more than initial 100 list

## License

[MIT](https://github.com/dakotahp/linkding-cleaner/blob/master/LICENSE)