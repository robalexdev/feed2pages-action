# Feed2Pages

A blogroll generator for [Hugo](https://gohugo.io/)-based websites that aggregates RSS feeds into your own news feed website.


## Use cases

* A personal RSS reader you can access from all your devices
* Share what you're reading and promote the RSS feeds you follow
* Aggregate multiple feeds to create a news feed for a particular topic
* Discover new blogs from the bloggers you follow


## Build your own

Please check out these example repos for help getting started:

* [feed2pages](https://github.com/ralexander-phi/feed2pages) - A generic quick start example
* [feed2pages-papermod](https://github.com/ralexander-phi/feed2pages-papermod) - A Hugo PaperMod example
* [Author's feeds](https://github.com/ralexander-phi/feeds) - The blogroll this project was built to generate


## Development

Build the utility:

    $ go build

In a directory with a feed.yaml file, run the utility:

    $ ./util


## How to promote your favorite feeds

Export an OPML from your feed reader.
Upload your OPML export as `https://<your-site>/.well-known/recommendations.opml` (or another location).
On each page of your website (or at least your home page) link to your OPML file using: `<link rel="blogroll" type="text/xml" href="https://<your-site>/.well-known/recommendations.opml">`.
Finally, edit your RSS feed to add the `<source:blogroll>` element.
See [blogroll.opml](https://opml.org/blogroll.opml) for more info.

For web-based readers, like [FeedLand](https://feedland.com), find a URL for your OPML file and link to that instead of uploading.

Software that blogroll discovery can help readers of your blog discover what you're reading.


## Discovering recommended feeds

A core feature of Feed2Pages is RSS feed recommendation discovery.

Various blogging engines support publishing a blogroll in a discoverable format.
These include:

* [Micro.blog](https://www.manton.org/2024/03/11/recommendations-and-blogrolls.html)
* [Ghost](https://ghost.org/docs/recommendations/)

These recommendations are the foundation of an RSS-based social network.
Feed2Pages walks this social network to aid discovery of connected websites.

For each feed in your OPML file, Feed2Pages will check for a `<source:blogroll>` element in the linked RSS feed.
The `.well-known/recommendations.opml` path of the website linked in the feed is also checked.
When one exists Feed2Pages will collect the information about each linked feed.
This process can continue iteratively to collect not only the recommended feeds of the feeds you follow, but the recommendations of those feeds as well.


## feeds.yaml settings

`feed_url`: The URL of your RSS OPML file.

If you're serving your OPML file with Hugo, set this to `file://static/.well-known/recommendations.opml`.

If you are using a site like [FeedLand](https://feedland.com), your subscriptions are available at `https://feedland.com/opml?screenname=<yourname>`.


### Post filters and limits

`post_age_limit_days`: Filter out posts older than this limit

`max_posts_per_feed`: Include only the newest N posts from each feed. This helps when some feeds publish content much more frequently than others, as they could otherwise fill the news feed.

`max_posts`: Limit the number of posts to display. Used for performance reasons.

`block_words`: Articles that contain any of these words in the title, description, or page content will be filtered out.

`block_domains`: Articles from this domain, or subdomains of this domain, will be filtered out.

`block_posts`: Individual posts that match these titles, GUIDs, or link URLs will be filtered out.


### Recommended feed discovery

`discover_depth`: How many iterations to perform when discovering recommended feeds. (default: 1. I.E. just the recommendations of the feeds you directly follow).

Set to zero to disable feed discovery.

`max_recommendations_per_feed`: How many recommendations to process per feed. (default: 100).

`max_recommendations`: How many recommendations to process in total (default: 1000).


### Configure output

`reading_folder_name`: Which content folder to store discovered posts. Default: reading

`following_folder_name`: Which content folder to store your followed feeds. Default: following

`discover_folder_name`: Which content folder to store feeds recommended by feeds you follow. Default: discover



## How it works

1. The repository owner configures the RSS feeds they wish to follow in `feeds.yaml`.
2. They configure settings such as block words to curate their news feed
3. GitHub Actions runs as a periodic (daily) cron job:
    1. The scraping utility collects articles from the RSS feeds
    2. The feed contents are normalized and enriched
    3. The discovered feeds and posts are saved as Hugo content
    4. Recommended feeds are discovered iteratively
    5. Hugo builds the site into static HTML
    6. GitHub Actions publishes the HTML to GitHub Pages
