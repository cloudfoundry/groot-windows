FROM microsoft/nanoserver:1709

USER Administrator
RUN mkdir C:\temp\test & echo hello > C:\temp\test\hello
RUN mklink C:\temp\symlinkfile C:\temp\test\hello
RUN mklink /H C:\temp\hardlinkfile C:\temp\test\hello
RUN mklink /D C:\temp\symlinkdir C:\temp\test
RUN mklink /J C:\temp\junctiondir C:\temp\test
