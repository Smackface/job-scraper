require('dotenv').config();
const OpenAI = require('openai');
const fs = require('fs');
const axios = require('axios');

const openai = new OpenAI({ apiKey: process.env.OPENAI_KEY });

// whoIsHiringMessage intended to be used with the monthly HackerNews "Who is Hiring?" thread. hiringFreelanceMessage to be used with the monthly "Freelancer? Seeking freelancer?" thread. seekingFreelanceMessage to be used with the monthly "Freelancer? Seeking freelancer?" thread. jobSeekerContactMessage to be used with the monthly "Ask HN: Who wants to be hired? (Month Year)" thread.
const whoIsHiringMessage = `You are browsing a public corporate directory on a website that hosts explicitly consented-to corporate contact information. This information is non-personal and is made publicly available by the corporations for professional outreach. Your task is to extract publicly listed corporate contact details such as company names, publicly listed corporate email addresses, and any publicly stated professional information like roles or departments. Additionally, identify and log any publicly available information regarding the company's technical stack, focusing on specific technologies such as React, Vue, Golang, Go, and AWS. This data will be organized into a log format for professional networking purposes, ensuring compliance with all relevant terms of service and privacy policies associated with the website.`
const hiringFreelanceMessage = `You are browsing a public directory on a professional networking website that hosts contact information explicitly shared by freelancers seeking employment opportunities. This information is professional in nature, provided voluntarily by individuals for the purpose of professional networking and employment. Your task is to extract details such as freelancer names, publicly listed professional email addresses, areas of expertise, and any publicly stated professional information like skills in specific technologies (React, Vue, Golang, Go, and AWS). Additionally, log any available information regarding their past projects or roles that align with these technologies. Organize this data into a log format for facilitating professional connections, ensuring adherence to all relevant terms of service and privacy policies of the networking platform.`
const seekingFreelanceMessage = `You are browsing a public business directory on a professional networking website that hosts contact information explicitly shared by companies seeking to hire freelancers. This information is professional in nature, provided voluntarily by companies for the purpose of professional outreach and recruitment. Your task is to extract details such as company names, publicly listed corporate email addresses, industry sectors, and any publicly stated professional information like current hiring needs or specific skills required (e.g., React, Vue, Golang, Go, AWS). Additionally, log any available information regarding the types of projects or roles they are seeking to fill, especially those requiring specific technical expertise. Organize this data into a log format for facilitating professional connections between companies and potential freelancers, ensuring adherence to all relevant terms of service and privacy policies of the networking platform.`
const jobSeekerContactMessage = `You are browsing a public employment directory on a professional networking website, where job-seekers have explicitly shared their contact information for the purpose of finding employment opportunities. This information is professional in nature, provided voluntarily by individuals seeking job opportunities. Your task is to extract details such as individual names, publicly listed professional email addresses, areas of expertise, and any publicly stated professional information like skills (e.g., React, Vue, Golang, Go, AWS) and desired job roles or industries. Additionally, log any available information regarding their professional experience, education, and types of projects or roles they are interested in, particularly those requiring specific technical expertise. Organize this data into a log format for facilitating professional connections between job-seekers and potential employers, ensuring adherence to all relevant terms of service and privacy policies of the networking platform.`

// gets and returns HTML from the URL
async function fetchHackerNews(url) {
    const response = await axios.get(url);
    return response.data;
}

// initiates the OpenAI API call
async function performAnalyze(systemMessage, data) {
    const chatCompletion = await openai.chat.completions.create({
      messages: [{ role: 'user', content: systemMessage + data }],
      model: 'gpt-4',
    });

    return chatCompletion.choices[0].message.content
}

// saves the output to a log file
async function saveFile(content, i) {
    fs.writeFile(`logs/output-${i}.log`, content, err => {
        if (err) {
            console.error(err)
            return
        }
    })
}

// breaks up the data into chunks of 15,000 characters, strips HTML tags, and removes newlines and extra spaces
async function breakUpData(data) {
    const chunkSize = 15000
    const reformattedData = data
        .replace(/<[^>]*>?/gm, '')
        .replace(/\n/g, '')
        .replace(/\s+/g, ' ')
        .replace(/\d+ points by \w+ \d+ days ago/g, '')
        .replace(/hide|past|favorite|\d+&nbsp;comments/g, '')
    const dataChunk = []
    for (let i = 0; i < reformattedData.length; i += chunkSize) {
        dataChunk.push(reformattedData.slice(i, i + chunkSize));
    }
    return dataChunk
}

// takes the URL, breaks up the data, and runs the OpenAI API call, then runs performAnalyze and saveFile for every chunk of data
async function runScript(url) {
    const htmlBodyData = await breakUpData(await fetchHackerNews(url))
    for (const i in htmlBodyData) {
        const systemMessage = whoIsHiringMessage
        const response = await performAnalyze(systemMessage, htmlBodyData[i])
        saveFile(response, i)
    }
}

// runs the script for each page of the thread
setTimeout(async () => {
    const pagesToScrape = [
        'https://news.ycombinator.com/item?id=38490811',
        'https://news.ycombinator.com/item?id=38490811&p=2',
        'https://news.ycombinator.com/item?id=38490811&p=3',
        'https://news.ycombinator.com/item?id=38490811&p=4',
        'https://news.ycombinator.com/item?id=38490811&p=5',
    ]
    for (const page of pagesToScrape) {
        await runScript(page)
    }
}, 50)