---
title: Branding
description: Give an instance its own name, logo, accent colour and VS Code color theme, so it looks like your own tool.
sidebar:
  order: 4
---

Branding makes an instance look like its owner's own tool, which is what a host wants when running Nexul for a client. It is an enterprise feature, so it needs an [enterprise license](/docs/guide/enterprise-license/) key that includes **Branding**. Everything it changes is the look: how the app behaves is the same.

Open **Settings → Instance**. The **Branding** card sits under **License**. Only someone whose role holds the instance's write permission, the instance Owner by default, can change it; anyone who can see the section sees what is set.

## Name and logo

- **Product name** replaces "Nexul" in the sidebar, on the sign-in page and the invitation page, in the browser tab and in the title of every phone notification the instance sends. It can be 40 characters.
- **Logo** replaces the Nexul mark in the sidebar and on sign-in, and is the browser tab's icon. Use a PNG, JPEG, WebP or SVG up to 512 KB; a square image fits best. A logo saves as soon as you choose it.

An instance with a product name drops Nexul's own pitch from its signed-out home page and keeps one small **Runs on Nexul** line on the sign-in pages.

The phone app keeps its own name and icon, since they are fixed when the app is built.

## Accent

The accent is the one colour for actions, the current page and selection. Pick it with the colour field and save. If it would be hard to read, as text or as a button label, Nexul moves its lightness a little and keeps its hue. **Use the default** goes back to Nexul's.

A person who has picked a colour theme of their own under **Settings → Appearance** keeps that theme's accent.

## A color theme for the whole instance

Under **Dark theme** and **Light theme**, **Import** a VS Code color theme file (a `*-color-theme.json` file; comments in it are fine). A theme file holds colours for one appearance, so a dark theme paints dark mode and a light theme paints light mode; the other mode keeps Nexul's colours until you import a theme for it.

Nexul takes the theme's background, text, secondary text, accent and error colours, and builds the panels, cards and borders between them the same way its own themes are built. Text must stay readable, so a colour that falls short of the contrast Nexul's own themes keep is moved until it holds, and the card says which: for example "Moved to stay readable: text, accent". The theme replaces every person's own theme in that mode, and **Settings → Appearance** says so. **Remove** drops it and goes back to Nexul's colours.

A theme file that includes another file is refused: upload one that holds its own colours.

## Without a key

When the key expires, is removed, or stops fitting the instance's address, the instance shows Nexul's own name, logo and colours again. What you saved stays, and shows again as soon as a key that includes Branding is in place. While the key is missing, the card keeps your settings but won't change them. **Remove branding** deletes them for good.

## From an agent

`instance_get` returns the branding with the rest of the instance's state. `instance_update` takes a `branding` object: `product_name`, `accent` (`#rrggbb`), `logo` (a base64 data URL), `theme_file` (the text of a theme file), `remove_theme` (`light`, `dark` or `all`) and `clear`. An empty `product_name`, `accent` or `logo` removes that setting. It needs the same permissions as the web page, and every open page follows the change at once.
