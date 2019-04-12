
---
title: "Getting Started"
linkTitle: "Getting Started"
weight: 3
date: 2018-07-30
description: >
  This page describes how to use this theme: How to install it, how to configure it, and the different components it contains.
resources:
- src: "**spruce*.jpg"
  params:
    byline: "Photo: Bjørn Erik Pedersen / CC-BY-SA"
---

Welcome to the Docsy theme user guide! This guide shows you how to get started creating technical documentation sites using Docsy, including site customization and how to use Docsy's blocks and templates.

## Installation and prerequisites 

You need a [recent version](https://github.com/gohugoio/hugo/releases) of Hugo to build and run sites (like this one) that use Docsy locally. If you install from the release page, make sure to get the `extended` Hugo version, which supports SCSS: you may need to scroll down the list of releases. Hugo can be installed via Brew if you're running MacOs. If you're a Linux user, do not use `sudo apt-get install hugo`, as it currently doesn't get you the `extended` version.

If you want to do stylesheet changes, you will also need `PostCSS` to create the final assets:

```
npm install -D --save autoprefixer
npm install -D --save postcss-cli
```

You can also install these tools globally on your computer:

```bash
npm install -g postcss-cli
npm install -g autoprefixer
```

To use a local version of the theme files during site development, clone the repo using:

```
git clone --recurse-submodules --depth 1 https://github.com/google/docsy.git
```

For comprehensive Hugo documentation, see [gohugo.io](https://gohugo.io/)

## Using the theme

To use the Docsy Hugo theme, you can either:

* Copy and edit this example site's repo, which will also give you a skeleton structure for your top-level and documentation sections.
* Specify the [Docsy theme](https://github.com/google/docsy) like [any other Hugo theme](https://gohugo.io/themes/installing-and-using-themes/) when creating or updating your site. This gives you all the theme-y goodness but you'll need to specify your own site structure.

## Configuring your site

See the examples with comments in `config.toml` in this project for how to add your project name, community links, configure Google Analytics, and so on. We recommend copying this `config.toml` and editing it even if you're just using the theme and not copying the entire Docsy example site.

## Content sections

The theme comes with templates for the top level sections `docs`, `blog`, and `community`, and a default landing page type of template used for any other section (in our example site it's used for the site landing page and the About page). See the pages in this site for examples of how to use the templates. For example, this page is in the `docs` folder so Hugo automatically applies the `docs` layout, which includes a left nav, page contents, and GitHub links (populated from your site config) for readers to edit the page or create issues.

The `community` landing page template has boilerplate content that's automatically filled in with the project name and community links specified in `config.toml`.

You can find out much more about how Hugo page layouts work in [Hugo Templates](https://gohugo.io/templates/).

### RSS feeds

Hugo will, by default, create an RSS feed for the home page and any section. For the main RSS feed you can control which sections to include by setting a site param in your `config.toml`. This is the default configuration:

```toml
rss_sections = ["blog"]
```
## Customizing landing pages.

If you've copied this example site, you already have a simple site landing page made up of Docsy's provided Hugo shortcode [page blocks](#shortcode-blocks) in `content/en/_index.html`. To customize the large landing image, which is in a [cover](#blocks-cover) block, replace the `content/en/featured-background.jpg` file in your project with your own image (it can be called whatever you like as long as it has `background` in the file name). You can remove or add as many blocks as you like, as well as adding your own custom content. 

The example site also has an About page in `content/en/about/_index.html` using the same Docsy landing page template. Again, this is made up of [page blocks](#shortcode-blocks), including another background image in `content/en/about/featured-background.jpg`. As with the site landing page, you can replace the image, remove or add blocks, or just add your own content.

If you've just used the theme, you can still use all the provided page blocks (or any other content you want) to build your own landing pages in the same file locations.

Find out more about using Docsy's page blocks in [Shortcode Blocks](#shortcode-blocks) below.

## Configuring navigation

### Top-level menu

Add a page or section to the top level menu by adding it to the `main` menu in either `config.toml` or in page front matter (in `_index.md` or `_index.html` for a section, as that's the section landing page). The menu is ordered by page `weight`:

```yaml
menu:
  main:
    weight: 20
```

So, for example, a section index or page with `weight: 30` would appear after this page in the menu, while one with `weight: 10` would appear before it.

### Section menu

The section menu, as shown in the left side of the `docs` section, is automatically built from the content tree. Like the top-level menu, it is ordered by page or section index `weight` (or by page creation `date` if `weight` is not set).

To hide a page or section from the menu, set `toc_hide: true` in front matter.

By default, the section menu will show the current section fully expanded all the way down. This may make the left nav too long and difficult to scan for bigger sites. Try setting site param `ui.sidebar_menu_compact = true` in `config.toml`.

### Breadcrumb navigation

Breadcrumb navigation is enabled by default. To disable breadcrumb navigation, set site param `ui.breadcrumb_disable = true` in `config.toml`.

## Changing the look and feel

### Color palette and other styles 

To quickly change your site's colors, add SCSS variable project overrides to `assets/scss/_variables_project.scss`. A simple example changing the primary and secondary color to two shades of purple:

```scss
$primary: #390040;
$secondary: #A23B72;
```

* See `assets/scss/_variables.scss` in the theme for color variables etc. that can be set to change the look and feel.
* Also see available variables in Bootstrap 4: https://getbootstrap.com/docs/4.0/getting-started/theming/ and https://github.com/twbs/bootstrap/blob/v4-dev/scss/_variables.scss

The theme has features suchs as rounded corners and gradient backgrounds enabled by default. These can also be toggled in your project variables file:

```scss
$enable-gradients: true;
$enable-rounded: true;
$enable-shadows: true;
```

{{% alert title="Tip" %}}
PostCSS (autoprefixing of CSS browser-prefixes) is not enabled when running in server mode (it is a little slow), so Chrome is the recommended choice for development.
{{% /alert %}}

Also note that any SCSS import will try the project before the theme, so you can -- as one example -- create your own `_assets/scss/_content.scss` and get full control over how your Markdown content is styled.

### Fonts

The theme uses [Open Sans](https://fonts.google.com/specimen/Open+Sans) as its primary font. To disable Google Fonts and use a system font, set this SCSS variable:

```scss
$td-enable-google-fonts: false;
```

To configure another Google Font:

```scss
$google_font_name: "Open Sans";
$google_font_family: "Open+Sans:300,300i,400,400i,700,700i";
```

Note that if you decide to go with a font with different weights (in the built-in configuration this is `300` (light), `400` (medium) and `700` (bold)), you also need to adjust the weight related variables, i.e. variables starting with `$font-weight-`.


## Custom shortcodes

### Shortcode blocks

The theme comes with a set of custom  **Page Blocks** as [Hugo Shortcodes](https://gohugo.io/content-management/shortcodes/) that can be used to compose landing pages, about pages and similar.

These blocks share some common parameters:

height
: A pre-defined height of the block container. One of `min`, `med`, `max`, `full`, or `auto`. Setting it to `full` will fill the Viewport Height, which can be useful for landing pages.

color
: The block will be assigned a color from the theme palette if not provided, but you can set your own if needed. You can use all of Bootstrap's color names, theme color names or a grayscale shade. Some examples would be `primary`, `white`, `dark`, `warning`, `light`, `success`, `300`, `blue`, `orange`. This will become the **bakground color** of the block, but text colors will adapt to get proper contrast.

#### blocks/cover

The **blocks/cover** shortcode is meant to create a landing page type of block that fills the top of the page.

```html
{{</* blocks/cover title="Welcome!" image_anchor="center" height="full" color="primary" */>}}
<div class="mx-auto">
	<a class="btn btn-lg btn-primary mr-3 mb-4" href="{{</* relref "/docs" */>}}">
		Learn More <i class="fas fa-arrow-alt-circle-right ml-2"></i>
	</a>
	<a class="btn btn-lg btn-secondary mr-3 mb-4" href="https://example.org">
		Download <i class="fab fa-github ml-2 "></i>
	</a>
	<p class="lead mt-5">This program is now available in <a href="#">AppStore!</a></p>
	<div class="mx-auto mt-5">
		{{</* blocks/link-down color="info" */>}}
	</div>
</div>
{{</* /blocks/cover */>}}
```

Note that the relevant shortcode parameters above will have sensible defaults, but is included here for completeness.

{{% alert title="Hugo Tip" %}}
> Using the bracket styled shortcode delimiter, `>}}`, tells Hugo that the inner content is HTML/plain text and needs no further processing. Changing it to `%}}` will treat it as Markdown. These can be mixed.
{{% /alert %}}


| Parameter        | Default    | Description  |
| ---------------- |------------| ------------|
| title | | The main display title for the block. | 
| image_anchor | |
| height | | See above.
| color | | See above. 


To set the background image, place an image with the word "background" in the name inside the [Page Bundle](https://gohugo.io/content-management/page-bundles/).

{{% alert title="Tip" %}}
If you also include the word **featured** in the image name, e.g. `my-featured-background.jpg`, it will also be used as the Twitter Card image when shared.
{{% /alert %}}

For available icons, see [Font Awesome](https://fontawesome.com/icons?d=gallery&m=free).

#### blocks/lead

The **blocks/lead** block shortcode is a simple lead/title block with centred text and an arrow down pointing to the next section.

```go-html-template
{{%/* blocks/lead color="dark" */%}}
TechOS is the OS of the future. 

Runs on **bare metal** in the **cloud**!
{{%/* /blocks/lead */%}}
```

| Parameter        | Default    | Description  |
| ---------------- |------------| ------------|
| height | | See above.
| color | | See above. 

#### blocks/section

The **blocks/section** shortcode is meant as a general-purpose content container. The example below shows it wrapping 3 feature sections.


```go-html-template
{{</* blocks/section color="dark" */>}}
{{%/* blocks/feature icon="fa-lightbulb" title="Fastest OS **on the planet**!" */%}}
The new **TechOS** operating system is an open source project. It is a new project, but with grand ambitions.
Please follow this space for updates!
{{%/* /blocks/feature */%}}
{{%/* blocks/feature icon="fab fa-github" title="Contributions welcome!" url="https://github.com/gohugoio/hugo" */%}}
We do a [Pull Request](https://github.com/gohugoio/hugo/pulls) contributions workflow on **GitHub**. New users are always welcome!
{{%/* /blocks/feature */%}}
{{%/* blocks/feature icon="fab fa-twitter" title="Follow us on Twitter!" url="https://twitter.com/GoHugoIO" */%}}
For announcement of latest features etc.
{{%/* /blocks/feature */%}}
{{</* /blocks/section */>}}
```

| Parameter        | Default    | Description  |
| ---------------- |------------| ------------|
| height | | See above.
| color | | See above. 


#### blocks/feature

```go-html-template

{{%/* blocks/feature icon="fab fa-github" title="Contributions welcome!" url="https://github.com/gohugoio/hugo" */%}}
We do a [Pull Request](https://github.com/gohugoio/hugo/pulls) contributions workflow on **GitHub**. New users are always welcome!
{{%/* /blocks/feature */%}}

```

| Parameter        | Default    | Description  |
| ---------------- |------------| ------------|
| title | | The title to use.
| url | | The URL to link to.
| icon | | The icon class to use.


#### blocks/link-down

The **blocks/link-down** shortcode creates a navigation link down to the next section. It's meant to be used in combination with the other blocks shortcodes.

```go-html-template

<div class="mx-auto mt-5">
	{{</* blocks/link-down color="info" */>}}
</div>
```

| Parameter        | Default    | Description  |
| ---------------- |------------| ------------|
| color | info | See above. 

### Shortcode helpers

####  alert

THe **alert** shortcode creates an alert block that can be used to display notices or warnings.

```go-html-template
{{%/* alert title="Warning" color="warning" */%}}
This is a warning.
{{%/* /alert */%}}

```

Renders to:

{{% alert title="Warning" color="warning" %}}
This is a warning.
{{% /alert %}}

| Parameter        | Default    | Description  |
| ---------------- |------------| ------------|
| color | primary | One of the theme colors, eg `primary`, `info`, `warning` etc.


####  imgproc

The **imgproc** shortcode finds an image in the current [Page Bundle](https://gohugo.io/content-management/page-bundles/) and scales it given a set of processing instructions.


The example above has also a byline with photo attribution added. When using illustrations with a free license from [WikiMedia](https://commons.wikimedia.org/) and simlilar, you will in most situations need a way to attribute the author or licensor. You can add metadata to your page resources in the page front matter. The `byline` param is used by convention in this theme:


```yaml
resources:
- src: "**spruce*.jpg"
  params:
    byline: "Photo: Bjørn Erik Pedersen / CC-BY-SA"
```


| Parameter        | Description  |
| ----------------: |------------|
| 1 | The image filename or enough of it to identify it (we do Glob matching)
| 2 | Command. One of `Fit`, `Resize` or `Fill`. See [Image Processing Methods](https://gohugo.io/content-management/image-processing/#image-processing-methods).
| 3 | Processing options, e.g. `400x450`. See [Image Processing Options](https://gohugo.io/content-management/image-processing/#image-processing-methods).


## CSS utilities

For documentation of available CSS utility classes, see the [Bootstrap Documentation](https://getbootstrap.com/). This theme adds very little on its own in this area. However, we have added some some color state CSS classes that can be useful in a dynamic context (when you don't know if the `primary` color is dark or light or you receive the color code as a shortcode parameter):

* `.-bg-<color>`
* `.-text-<color>`

The value of `<color>` can be any of the color names, `primary`, `white`, `dark`, `warning`, `light`, `success`, `300`, `blue`, `orange` etc.

For `.-bg-<color>`, the text colors will be adjusted to get proper contrast:

```html
<div class="-bg-primary p-3 display-4">Background: Primary</div>
<div class="-bg-200 p-3 display-4">Background: Gray 200</div>
```

<div class="-bg-primary p-3 display-4 w-75">Background: Primary</div>
<div class="-bg-200 p-3 display-4 mb-5 w-50 w-75">Background: Gray 200</div>

`.-text-<color>` sets the text color only:

```html
<div class="-text-blue pt-3 display-4">Text: Blue</div>
```

<div class="-text-blue pt-3 display-4">Text: Blue</div>


## Multilanguage support

### Navigation

If you configure more than one language in `config.toml`, a language selector will be added to the top-level menu. It will take you to the translated version of the current page, or the home page for the given language.

### i18n bundles

All UI strings (text for buttons etc.) are bundled inside `/i18n` in the theme. Translations (e.g. create a copy of `en.toml` to `jp.toml`) should be done in the theme, so it can be reused by others. Additional strings or overridden values can be added to the project's `/i18n` folder.

{{% alert title="Hugo Tip" %}}
Run `hugo server --i18n-warnings` when doing translation work, as it will give you warnings on what strings are missing.
{{% /alert %}}

### Content

For `content`, each language can have its own language configuration and its own content root, e.g. `content/en`. See the [Hugo Docs](https://gohugo.io/content-management/multilingual) on multi-language support for more information.

## Add your logo

Add your project logo to `assets/icons/logo.svg` in your project.

## Add your favicons

The easiest way to do this is to create a set of favicons via http://cthedot.de/icongen (which lets you create a huge range of icon sizes and options from a single image) and/or https://favicon.io/, and put them in your site project's `static/favicons` directory. This will override the default favicons from the theme.

Note that https://favicon.io/  doesn't create as wide a range of sizes as Icongen but *does* let you quickly create favicons from text: if you want to create text favicons you can use this site to generate them, then use Icongen to create more sizes (if necessary) from your generated `.png` file.

If you have special favicon requirements, you can create your own `layouts/partials/favicons.html` with your links.

## Configure search

1. Add your Google Custom Search Engine ID to the site params in `config.toml`. You can add different values per language if needed.
2. Add a content file in `content/en/search.md` (and one per other languages if needed). It only needs a title and `layout: search`.

## Customizing templates

### Add code to head or before body end

If you need to add some code (CSS import or similar) to the `head` section on every page, add a partial to your project:

```
layouts/partials/hooks/head-end.html
```

And add the code you need in that file.

Similar, if you want to add some code right before the `body` end:

```
layouts/partials/hooks/body-end.html
```

## Deploying your site

There are multiple possible options for deploying a Hugo site; you can read about them all in [Hosting and Deployment](https://gohugo.io/hosting-and-deployment/). 

### Deployment with Netlify

We recommend using [Netlify](https://www.netlify.com/) as a particularly simple way to serve your site from GitHub, with continuous deployment from GitHub, previews, and more. Netlify is free to use for Open Source projects, with premium tiers if you require greater support.

Follow the instructions in [Host on Netlify](https://gohugo.io/hosting-and-deployment/hosting-on-netlify/) to deploy your site. Specify at least the 0.47 version of Hugo when configuring your deployment as earlier versions won't work with this theme.

{{% alert title="Warning" color="warning" %}}
At the moment due to Netlify system limitations, Netlify does not support the "extended" version of Hugo needed to use SCSS, which is used by our theme. This is a known issue and the fix will be rolled out in future versions of Netlify. A workaround until then is to build the site on your local machine with "extended" Hugo, and then commit the generated `resources/` folder to your site repo on GitHub.  To do this:

1.  Ensure you have an up to date local copy of your site files cloned from your repo. Don't forget to use `--recurse-submodules` or you won't pull down some of the code you need to generate a working site.

    ```
    git clone --recurse-submodules --depth 1 https://github.com/my/example.git
    ```

1.  Ensure you have the tools described in [Installation and Prerequisites](#installation-and-prerequisites) installed on your local machine, including `postcss-cli`: you'll need it to generate the site resources.
1.  Run the `hugo` command in your site root.
1.  Add the generated `resources/` directory using `git add -f resources`, and commit back to the repo.

You should now be able to serve the complete site from GitHub using Netlify. Please check our docs for updates on when you will no longer need this workaround.
{{% /alert %}}

### Serving your site locally

Depending on your deployment choice you may want to serve your site locally during development to preview content changes. To serve your site locally:

1.  Ensure you have an up to date local copy of your site files cloned from your repo. Don't forget to use `--recurse-submodules` or you won't pull down some of the code you need to generate a working site.

    ```
    git clone --recurse-submodules --depth 1 https://github.com/my/example.git
    ```
   
    {{% alert title="Note" color="primary" %}}
If you've just added the theme as a submodule in a local version of your site and haven't committed it to a repo yet,  you must get local copies of the theme's own submodules before serving your site.
    
    git submodule update --init --recursive
    {{% /alert %}}

1.  Ensure you have the tools described in [Installation and Prerequisites](#installation-and-prerequisites) installed on your local machine, including `postcss-cli` (you'll need it to generate the site resources the first time you run the server).
1.  Run the `hugo server` command in your site root.

## Keeping the theme up to date

We hope to continue to make improvements to the theme along with the Docsy community. If you have cloned the example site (or are otherwise using the theme as a submodule), you can update the theme submodule yourself as follows:

1. In your local copy of your project, run:

    ```
    git submodule update --remote
    ```
    
1. Then add and commit your change:

    ```
    git add themes/
    git commit -m "Updating theme submodule"
    ```

1. Finally, push the change back to the project repo.

    ```
    git push origin master
    ```
    
If you've cloned the theme yourself, use `git pull origin master` in the theme root directory to get the latest version.

## Images used on this site

Images used as background images in this example site are in the [public domain](https://commons.wikimedia.org/wiki/User:Bep/gallery#Wed_Aug_01_16:16:51_CEST_2018) and can be used freely.



	
