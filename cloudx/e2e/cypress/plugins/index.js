/// <reference types="cypress" />
// ***********************************************************
// This example plugins/index.js can be used to load plugins
//
// You can change the location of this file or turn off loading
// the plugins file with the 'pluginsFile' configuration option.
//
// You can read more here:
// https://on.cypress.io/plugins-guide
// ***********************************************************

// This function is called when a project is opened or re-opened (e.g. due to
// the project's config changing)
const jwks = require('jwks-rsa')
const jwt = require('jsonwebtoken')

function getKey(header, callback) {
  client.getSigningKey(header.kid, function (err, key) {
    var signingKey = key.publicKey || key.rsaPublicKey
    callback(null, signingKey)
  })
}

/**
 * @type {Cypress.PluginConfig}
 */
// eslint-disable-next-line no-unused-vars
module.exports = (on, config) => {
  // `on` is used to hook into various events Cypress emits
  // `config` is the resolved Cypress config
  on('task', {
    verify(token) {
      return new Promise((resolve, reject) => {
        const client = jwks({
          cache: false,
          jwksUri: config.baseUrl + '/.ory/proxy/jwks.json'
        })

        jwt.verify(
          token,
          (header, callback) => {
            client.getSigningKey(header.kid, (err, key) => {
              callback(null, key.publicKey || key.rsaPublicKey)
            })
          },
          undefined,
          (err, decoded) => {
            if (err) {
              return reject(err)
            }

            return resolve(decoded)
          }
        )
      })
    }
  })
}
