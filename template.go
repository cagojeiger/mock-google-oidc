package main

import "html/template"

var loginTemplate = template.Must(template.New("login").Parse(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>Test IDP</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f5f5f5; display: flex; justify-content: center; padding-top: 60px; }
  .card { background: #fff; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); padding: 40px; width: 420px; }
  h1 { font-size: 24px; margin-bottom: 8px; }
  h2 { font-size: 14px; color: #666; margin-bottom: 24px; font-weight: normal; }
  label { display: block; font-size: 14px; font-weight: 500; margin-bottom: 4px; color: #333; }
  input[type="text"], input[type="email"] { width: 100%; padding: 10px 12px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; margin-bottom: 16px; }
  input[type="text"]:focus, input[type="email"]:focus { outline: none; border-color: #4285f4; }
  .btn { display: block; width: 100%; padding: 12px; background: #4285f4; color: #fff; border: none; border-radius: 4px; font-size: 16px; cursor: pointer; margin-top: 8px; }
  .btn:hover { background: #3367d6; }
  details { margin-top: 20px; }
  summary { cursor: pointer; font-size: 13px; color: #666; }
  .radio-group { margin: 12px 0; }
  .radio-group label { display: inline-block; margin-right: 16px; font-weight: normal; }
  .debug { margin-top: 20px; padding: 12px; background: #f9f9f9; border-radius: 4px; font-size: 12px; color: #888; word-break: break-all; }
  .debug div { margin-bottom: 4px; }
  .debug span { color: #333; }
</style>
</head>
<body>
<div class="card">
  <h1>Test IDP</h1>
  <h2>Sign in to continue</h2>
  <form method="POST" action="/o/oauth2/v2/auth">
    <input type="hidden" name="redirect_uri" value="{{.RedirectURI}}">
    <input type="hidden" name="state" value="{{.State}}">
    <input type="hidden" name="nonce" value="{{.Nonce}}">
    <input type="hidden" name="scope" value="{{.Scope}}">
    <input type="hidden" name="client_id" value="{{.ClientID}}">
    <input type="hidden" name="code_challenge" value="{{.CodeChallenge}}">
    <input type="hidden" name="code_challenge_method" value="{{.CodeChallengeMethod}}">

    <label for="email">Email</label>
    <input type="email" id="email" name="email" value="{{.Email}}" required>

    <label for="name">Name</label>
    <input type="text" id="name" name="name" value="{{.Name}}" required>

    <button type="submit" class="btn">Login</button>

    <details>
      <summary>Advanced (Response Mode)</summary>
      <div class="radio-group">
        <label><input type="radio" name="response_mode" value="normal" checked> Normal</label>
        <label><input type="radio" name="response_mode" value="deny"> Deny</label>
        <label><input type="radio" name="response_mode" value="token_error"> Token Error</label>
        <label><input type="radio" name="response_mode" value="userinfo_error"> Userinfo Error</label>
      </div>
    </details>
  </form>

  <div class="debug">
    <div>client_id: <span>{{.ClientID}}</span></div>
    <div>redirect_uri: <span>{{.RedirectURI}}</span></div>
    <div>scope: <span>{{.Scope}}</span></div>
    <div>state: <span>{{.State}}</span></div>
  </div>
</div>
</body>
</html>`))

// LoginPageData holds the data for the login page template.
type LoginPageData struct {
	RedirectURI         string
	State               string
	Nonce               string
	Scope               string
	ClientID            string
	CodeChallenge       string
	CodeChallengeMethod string
	Email               string
	Name                string
}
