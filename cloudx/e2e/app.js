const express = require('express')
const app = express()
const port = 4001

app.get('/', (req, res) => {
    res.send("app home")
})

app.get('/anything', (req, res) => {
    res.send({headers: req.headers})
})

app.listen(port, () => {
    console.log(`Example app listening on port ${port}`)
})
