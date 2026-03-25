/*
Part of exercise file for go lang course at
https://web.learncodeonline.in
*/

const express = require('express');
const app = express();

const PORT = 8000;

app.use(express.json());
app.use(express.urlencoded({ extended: true }));

// Route
app.get('/', (req, res) => {
    res.status(200).send('Welcome to Express server!');
});

app.get('/get', (req, res) => {
    res.status(200).json({ message: 'Hello from Express server!' });
})

app.post('/post', (req, res) => {
    let myJson = req.body;     // Your Json

    res.status(200).send(myJson);
})

app.post('/postform', (req, res) => {
    res.status(200).send(JSON.stringify(req.body));
})

app.listen(PORT, () => {
    console.log(`App listening at http://localhost:${PORT}`);
});