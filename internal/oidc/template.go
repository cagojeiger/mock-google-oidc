package oidc

import "html/template"

var loginTemplate = template.Must(template.New("login").Parse(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>Mock Google Login</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }

  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    background: #1e1e1e;
    color: #cccccc;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }

  /* ── Top bar ── */
  .topbar {
    background: #2d2d2d;
    border-bottom: 1px solid #404040;
    padding: 10px 24px;
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .topbar-title {
    font-size: 14px;
    font-weight: 600;
    color: #e0e0e0;
    font-family: "SF Mono", "Fira Code", Menlo, monospace;
  }
  .mock-badge {
    font-size: 11px;
    font-weight: 600;
    color: #f0b000;
    background: rgba(240,176,0,0.12);
    border: 1px solid rgba(240,176,0,0.3);
    border-radius: 4px;
    padding: 3px 10px;
    letter-spacing: 0.05em;
    text-transform: uppercase;
  }

  /* ── Main layout ── */
  .main {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 40px 24px;
  }

  .container {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0;
    width: 100%;
    max-width: 780px;
    background: #252526;
    border-radius: 10px;
    border: 1px solid #404040;
    overflow: hidden;
  }

  /* ── Left: login form ── */
  .panel-left {
    padding: 40px 36px;
    border-right: 1px solid #404040;
  }

  .panel-title {
    font-size: 20px;
    font-weight: 600;
    color: #e0e0e0;
    margin-bottom: 4px;
  }
  .panel-subtitle {
    font-size: 13px;
    color: #888;
    margin-bottom: 28px;
  }
  .panel-subtitle .client-name {
    color: #569cd6;
    font-weight: 500;
  }

  .form-group {
    margin-bottom: 18px;
  }
  .form-group label {
    display: block;
    font-size: 12px;
    font-weight: 500;
    color: #888;
    margin-bottom: 6px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .form-input {
    width: 100%;
    padding: 10px 12px;
    background: #1e1e1e;
    border: 1px solid #404040;
    border-radius: 6px;
    color: #e0e0e0;
    font-size: 14px;
    font-family: "SF Mono", "Fira Code", Menlo, monospace;
    outline: none;
    transition: border-color 0.15s;
  }
  .form-input:focus {
    border-color: #569cd6;
    box-shadow: 0 0 0 2px rgba(86,156,214,0.15);
  }

  .btn-signin {
    display: block;
    width: 100%;
    padding: 11px;
    background: #0e639c;
    color: #fff;
    border: none;
    border-radius: 6px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    margin-top: 24px;
    transition: background 0.15s;
  }
  .btn-signin:hover { background: #1177bb; }
  .btn-signin:active { background: #0d5689; }

  /* ── Right: debug panel ── */
  .panel-right {
    padding: 40px 28px;
    background: #1e1e1e;
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .section-title {
    font-size: 11px;
    font-weight: 600;
    color: #888;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    margin-bottom: 10px;
    padding-bottom: 6px;
    border-bottom: 1px solid #333;
  }

  .info-grid {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 5px 14px;
    font-size: 13px;
    font-family: "SF Mono", "Fira Code", Menlo, monospace;
  }
  .info-key {
    color: #569cd6;
    text-align: right;
    user-select: none;
  }
  .info-val {
    color: #ce9178;
    word-break: break-all;
  }

  /* ── Simulate pills ── */
  .radio-group {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
  .radio-pill { cursor: pointer; }
  .radio-pill input[type="radio"] { display: none; }
  .radio-pill span {
    display: inline-block;
    padding: 5px 14px;
    border-radius: 4px;
    font-size: 12px;
    font-family: "SF Mono", "Fira Code", Menlo, monospace;
    color: #aaa;
    background: #2d2d2d;
    border: 1px solid #404040;
    transition: all 0.12s;
  }
  .radio-pill input[type="radio"]:checked + span {
    background: rgba(86,156,214,0.15);
    color: #569cd6;
    border-color: #569cd6;
  }
  .radio-pill:hover span {
    border-color: #555;
  }
</style>
</head>
<body>

<div class="topbar">
  <span class="topbar-title">mock-google-oidc</span>
  <span class="mock-badge">Mock Mode</span>
</div>

<div class="main">
  <form class="container" method="POST" action="/o/oauth2/v2/auth">
    <input type="hidden" name="redirect_uri" value="{{.RedirectURI}}">
    <input type="hidden" name="state" value="{{.State}}">
    <input type="hidden" name="nonce" value="{{.Nonce}}">
    <input type="hidden" name="scope" value="{{.Scope}}">
    <input type="hidden" name="client_id" value="{{.ClientID}}">
    <input type="hidden" name="code_challenge" value="{{.CodeChallenge}}">
    <input type="hidden" name="code_challenge_method" value="{{.CodeChallengeMethod}}">

    <!-- Left: Login Form -->
    <div class="panel-left">
      <div class="panel-title">Sign in</div>
      <div class="panel-subtitle">to continue to <span class="client-name">{{if .ClientID}}{{.ClientID}}{{else}}app{{end}}</span></div>

      <div class="form-group">
        <label for="email">Email</label>
        <input class="form-input" type="email" id="email" name="email" value="{{.Email}}" required>
      </div>

      <div class="form-group">
        <label for="name">Display Name</label>
        <input class="form-input" type="text" id="name" name="name" value="{{.Name}}" required>
      </div>

      <button type="submit" class="btn-signin">Sign In</button>
    </div>

    <!-- Right: Debug Panel -->
    <div class="panel-right">
      <div>
        <div class="section-title">Request</div>
        <div class="info-grid">
          <span class="info-key">client</span><span class="info-val">{{.ClientID}}</span>
          <span class="info-key">redirect</span><span class="info-val">{{.RedirectURI}}</span>
          <span class="info-key">scope</span><span class="info-val">{{.Scope}}</span>
          <span class="info-key">state</span><span class="info-val">{{.State}}</span>
          {{if .Nonce}}<span class="info-key">nonce</span><span class="info-val">{{.Nonce}}</span>{{end}}
          {{if .CodeChallenge}}<span class="info-key">pkce</span><span class="info-val">{{.CodeChallengeMethod}}</span>{{end}}
        </div>
      </div>

      <div>
        <div class="section-title">Simulate Response</div>
        <div class="radio-group">
          <label class="radio-pill"><input type="radio" name="response_mode" value="normal" checked><span>Normal</span></label>
          <label class="radio-pill"><input type="radio" name="response_mode" value="deny"><span>Deny</span></label>
          <label class="radio-pill"><input type="radio" name="response_mode" value="token_error"><span>Token Error</span></label>
          <label class="radio-pill"><input type="radio" name="response_mode" value="userinfo_error"><span>Userinfo Error</span></label>
        </div>
      </div>
    </div>

  </form>
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
