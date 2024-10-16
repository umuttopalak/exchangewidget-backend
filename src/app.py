import os

from apscheduler.schedulers.background import BackgroundScheduler
from flask import Flask

app = Flask(__name__)

def scheduled_task():
    print({"data": "Hello Backend"})

@app.route('/')
def hello():
    return {"data": "Hello Backend"}

if __name__ == '__main__':
    scheduler = BackgroundScheduler()
    scheduler.add_job(scheduled_task, 'interval', minutes=10)
    scheduler.start()

    port = int(os.environ.get('PORT', 5000))
    app.run(host='0.0.0.0', port=port)
    
    scheduler.shutdown()
