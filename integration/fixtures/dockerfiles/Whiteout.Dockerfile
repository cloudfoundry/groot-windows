FROM microsoft/nanoserver:1709

RUN mkdir C:\temp\test & echo hello > C:\temp\test\hello
RUN del /f C:\temp\test\hello

RUN echo hello2 > C:\temp\test\hello2
