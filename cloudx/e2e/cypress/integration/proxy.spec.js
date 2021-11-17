const randomString = () => (Math.random() + 1).toString(36).substring(7)
const randomPassword = () => randomString() + randomString()
const randomEmail = () => randomString() + '@' + randomString() + '.com'


const login = (email, password) => {
  cy.visit('/.ory/ui/login')
  cy.get('[name="password_identifier"]').type(email)
  cy.get('[name="password"]').type(password)
  cy.get('[name="method"]').click()
  loggedIn(email)
}

const loggedIn = (email) => {
  cy.visit('/.ory/ui/welcome')
  cy.get('pre code').should('contain.text', email)
  cy.get('[data-testid="logout"]').should('have.attr', 'aria-disabled', 'false')
}

describe('ory proxy', () => {
  const email = randomEmail()
  const password = randomPassword()
  before(() => {
    cy.clearCookies({domain: null})
  });

  it('navigation works', () => {
    cy.visit('/.ory/ui/registration');
    cy.get('.card-action a').click()
    cy.location('pathname').should('eq', '/.ory/ui/login')
  })

  it('should be able to execute registration', () => {
    cy.visit('/.ory/ui/registration');
    cy.get('[name="traits.email"]').type(email)
    cy.get('[name="password"]').type(password)
    cy.get('[name="method"]').click()
    cy.visit('/.ory/ui/welcome')
    loggedIn(email)
  })

  it('should be able to execute login', () => {
    login(email, password)
    cy.request('/anything').should((res) => {
      expect(res.body.headers['Authorization']).to.not.be.empty
      const token = res.body.headers['Authorization'].replace(/bearer /gi, '')
      console.log({token})

      cy.task('verify', res.body.headers['Authorization'].replace(/bearer /gi, '')).then((decoded) => {
        expect(decoded.session.identity.traits.email).to.equal(email)
      })
    });
  })

  it('should be able to execute logout', () => {
    login(email, password)
    cy.visit('/.ory/ui/welcome')
    cy.get('[data-testid="logout"]').should('have.attr', 'aria-disabled', 'false')
    cy.get('[data-testid="logout"]').click()
    cy.request('/anything').should((res) => {
      expect(res.body.headers['Authorization']).to.be.undefined
    });
  })
})
