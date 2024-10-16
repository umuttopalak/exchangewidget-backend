from flask import Flask

app = Flask(__name__)

@app.route("/")
def healthcheck():
    return {"data": "Hello Backend"}

if __name__ == '__main__':
    app.run(debug=True)
