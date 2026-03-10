# <u>The Basic Principle of the Project</u>

I want to make a system that keeps the record of the things that I share, it could be any source like an image or PDF or document or any hyperlink. Based on those "sources" it will be classified into a category for example:

        Taking a "hyperlink" example:
        1) If I provide a hyperlink which leads to a website that tells that "There will be an internship after 5 months so stay tuned for more information", so the system classifies that hyperlink to a category of "internships" meaning the user has an intrest in that internship so he is saving that website hyperlink so that he can apply it in future when the time comes.

        2) If I provide a Hyperlike which is a short video, then the system would classify the link based on what video is, let's say that video is on a podcast about "Neural Network", so there will be another category where this is saved.

These sources are saved in form of a Graph (data structure) for example "internship" will be the parent node and the other "sources" attached to that parent node would be child nodes. (Yet we still need to work more on the logic for how these classification will be done.)

This system will be able to access the sources I provide, for example
        
        1) If I the system a "hyperlink" of a website which tells that there is a Hackathon on 25th january 2026 then it will be able to read through that webpage and sum up all the information on its own and extract the meaningfull things such as "Date of Hackathon", "Topic of Hackathon", "Deaadline to apply" etc. And use these information to make a todo-list type of thing or it could also be added in the calender as an "Event". Now at times the system will remind me about this hackathon.

        2) If I give the system a "hyperlink" of a video/clip then the system would be able to go through the video and will also be able to extract valuable informaiton.

        3) If I give the system a "Image" or "PDF" or any other file/document then the system will be able to scrap out valuable information.

[ Potential Problem: How would the system would know what information to extract?]
[ Potential Solution: The system would first analyse the source and will come up with his own assumption about What information to extract, it may also be possible that the user will also provide a context to extract information, so based no that source the system will extract the information.]

Then there will be a chat system where the user can interact with the shared resources. Interaction line chating or understanding about the shared resoures. They will also be able to search up thier shared resource for specific things, for example:

        Let's say we have 5 independent parent nodes for Technology, Marketing, Finance, Meida and AI. Each of these nodes will have many sub-nodes which will have resources that tey are categoried for. So let's say the user saved an article related to RAG, then this would fall into "AI category" meaning the "AI Parent Node" so if user searchs in "RAG" in the chat section thenthe source node(s) associated with RAG will get highlighted.

# <u>Architecture of the Project</u>

This "system" will function on Windows and Andriod. This system will be in form of an Application. Both the Windows app and the Android app will be seemlesly connected for the data they recieve.

## <u>Andriod Fetures</u>
        
        1) "Implicit Intent" active, the app will be able to handle all types of data (text/url, images or documents) when shared to it.


# <u>UI/UX of this project</u>

## Node Graph

The sources will be saved in form of "Node Graphs". This will be in an "interactive 3D environment node graph" and also an "interactive 2d plain node graph", the option will be given to the user for switching between these two environments. 

There will be a "Tree View Navigation Bar" for the "Node Graphs". These nodes will be color coded to avoid confusion.

On the extreme left there will be navigation pannel for multiple features that are provided. At right of "Navigation Pannel" there will be a pannel that will show the instances of the features. 


# <u>Tools/Resources to use</u>

1) The app should be MCP integratable.

2) The application's backed should be based on Golang